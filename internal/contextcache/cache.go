// Package contextcache keeps the small, derived Slack context that agents need
// to choose a channel without copying a changing channel list into a shared
// repository. It deliberately stores only identifiers and activity aggregates;
// message bodies and tokens never enter the cache.
package contextcache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

const (
	SchemaVersion      = 1
	DefaultWindowDays  = 30
	DefaultRefreshAge  = 24 * time.Hour
	DefaultMaxPages    = 100
	SearchPageSize     = 100
	lockWaitAttempts   = 50
	lockWaitInterval   = 100 * time.Millisecond
	lockStaleAfter     = 10 * time.Minute
	lockHeartbeatEvery = lockStaleAfter / 3
	cacheDirectoryMode = 0o700
	cacheFileMode      = 0o600
)

const (
	StatusFresh   = "fresh"
	StatusPartial = "partial"
	StatusStale   = "stale"
)

// SearchClient is the subset of slack.Client used to build the cache. Keeping
// this interface here makes the cache testable without a live Slack request.
type SearchClient interface {
	AuthTestContext(context.Context) (*slack.AuthTestResponse, error)
	SearchMessagesContext(context.Context, string, slack.SearchParameters) (*slack.SearchMessages, error)
}

type Options struct {
	CacheDir       string
	Now            func() time.Time
	WindowDays     int
	RefreshAge     time.Duration
	MaxSearchPages int
	Force          bool
}

type Workspace struct {
	ID string `json:"id"`
}

type User struct {
	ID string `json:"id"`
}

type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Days  int       `json:"days"`
}

type Channel struct {
	ID             string    `json:"id"`
	Name           string    `json:"name,omitempty"`
	IsPrivate      bool      `json:"is_private"`
	IsMPIM         bool      `json:"is_mpim"`
	ActivityCount  int       `json:"activity_count"`
	LastActivityAt time.Time `json:"last_activity_at"`
}

type Snapshot struct {
	SchemaVersion int       `json:"schema_version"`
	Workspace     Workspace `json:"workspace"`
	User          User      `json:"user"`
	Window        Window    `json:"window"`
	RefreshedAt   time.Time `json:"refreshed_at"`
	Partial       bool      `json:"partial"`
	Channels      []Channel `json:"channels"`
}

type Result struct {
	Snapshot     Snapshot
	Status       string
	Refreshed    bool
	RefreshError string
}

// Ensure returns a fresh snapshot when the cache is missing or older than the
// refresh age. A usable old snapshot is returned with status=stale when the
// refresh cannot complete; callers can continue while clearly reporting that
// the data is not current.
func Ensure(ctx context.Context, client SearchClient, opts Options) (Result, error) {
	if client == nil {
		return Result{}, errors.New("slack context: nil API client")
	}
	now := nowFunc(opts)().UTC()
	windowDays := opts.WindowDays
	if windowDays <= 0 {
		windowDays = DefaultWindowDays
	}
	refreshAge := opts.RefreshAge
	if refreshAge <= 0 {
		refreshAge = DefaultRefreshAge
	}
	maxPages := opts.MaxSearchPages
	if maxPages <= 0 {
		maxPages = DefaultMaxPages
	}

	auth, err := client.AuthTestContext(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("slack context auth.test: %w", err)
	}
	if auth == nil || auth.TeamID == "" || auth.UserID == "" {
		return Result{}, errors.New("slack context auth.test returned no workspace or user ID")
	}

	cachePath, err := cachePath(opts.CacheDir, auth.TeamID, auth.UserID)
	if err != nil {
		return Result{}, err
	}
	if err := ensureCacheDirectories(cachePath); err != nil {
		return Result{}, err
	}
	var old *Snapshot
	if cached, readErr := readSnapshot(cachePath); readErr == nil && cacheMatchesAuth(cached, auth) {
		old = cached
	}
	if old != nil && !opts.Force && !needsRefresh(old, now, refreshAge, windowDays) {
		return Result{Snapshot: *old, Status: statusForSnapshot(*old)}, nil
	}
	var result Result
	if err := withLock(ctx, cachePath+".lock", func(lockCtx context.Context) error {
		if cached, readErr := readSnapshot(cachePath); readErr == nil && cacheMatchesAuth(cached, auth) && !opts.Force && !needsRefresh(cached, now, refreshAge, windowDays) {
			result = Result{Snapshot: *cached, Status: statusForSnapshot(*cached)}
			return nil
		}

		old, _ = readSnapshot(cachePath)
		if !cacheMatchesAuth(old, auth) {
			old = nil
		}
		snapshot, refreshErr := buildSnapshot(lockCtx, client, auth, now, windowDays, maxPages)
		if refreshErr != nil {
			if old != nil {
				result = Result{
					Snapshot:     *old,
					Status:       "stale",
					RefreshError: ErrorKind(refreshErr),
				}
				return nil
			}
			return refreshErr
		}
		if err := lockCtx.Err(); err != nil {
			return err
		}
		if err := writeSnapshot(cachePath, snapshot); err != nil {
			if old != nil {
				result = Result{
					Snapshot:     *old,
					Status:       StatusStale,
					RefreshError: ErrorKind(err),
				}
				return nil
			}
			return err
		}
		result = Result{Snapshot: snapshot, Status: statusForSnapshot(snapshot), Refreshed: true}
		return nil
	}); err != nil {
		if old != nil {
			return Result{
				Snapshot:     *old,
				Status:       StatusStale,
				RefreshError: ErrorKind(err),
			}, nil
		}
		return Result{}, err
	}
	return result, nil
}

func statusForSnapshot(snapshot Snapshot) string {
	if snapshot.Partial {
		return StatusPartial
	}
	return StatusFresh
}

func nowFunc(opts Options) func() time.Time {
	if opts.Now != nil {
		return opts.Now
	}
	return time.Now
}

func needsRefresh(snapshot *Snapshot, now time.Time, refreshAge time.Duration, windowDays int) bool {
	if snapshot == nil || snapshot.SchemaVersion != SchemaVersion {
		return true
	}
	if snapshot.Window.Days != windowDays || snapshot.RefreshedAt.IsZero() {
		return true
	}
	return now.Sub(snapshot.RefreshedAt) >= refreshAge
}

func cacheMatchesAuth(snapshot *Snapshot, auth *slack.AuthTestResponse) bool {
	return snapshot != nil && auth != nil && snapshot.Workspace.ID == auth.TeamID && snapshot.User.ID == auth.UserID
}

func buildSnapshot(ctx context.Context, client SearchClient, auth *slack.AuthTestResponse, now time.Time, windowDays, maxPages int) (Snapshot, error) {
	start := now.Add(-time.Duration(windowDays) * 24 * time.Hour)
	// Slack's date filter is day-granular. Start one day earlier and apply the
	// exact timestamp filter below so activity on the first window day is not
	// lost at the boundary.
	query := fmt.Sprintf("from:<@%s> after:%s", auth.UserID, start.Add(-24*time.Hour).Format("2006-01-02"))
	params := slack.NewSearchParameters()
	params.Count = SearchPageSize
	params.Sort = "timestamp"
	params.SortDirection = "desc"

	byID := make(map[string]Channel)
	partial := false
	for page := 1; page <= maxPages; page++ {
		params.Page = page
		resp, err := client.SearchMessagesContext(ctx, query, params)
		if err != nil {
			return Snapshot{}, fmt.Errorf("slack context search.messages page %d: %w", page, err)
		}
		if resp == nil {
			return Snapshot{}, fmt.Errorf("slack context search.messages page %d returned no response", page)
		}
		for _, match := range resp.Matches {
			if match.Channel.ID == "" || match.User != auth.UserID {
				continue
			}
			timestamp, err := parseSlackTimestamp(match.Timestamp)
			if err != nil || timestamp.Before(start) || timestamp.After(now) {
				continue
			}
			channel := byID[match.Channel.ID]
			channel.ID = match.Channel.ID
			channel.Name = firstNonEmpty(match.Channel.Name, channel.Name)
			channel.IsPrivate = match.Channel.IsPrivate
			channel.IsMPIM = match.Channel.IsMPIM
			channel.ActivityCount++
			if channel.LastActivityAt.IsZero() || timestamp.After(channel.LastActivityAt) {
				channel.LastActivityAt = timestamp
			}
			byID[channel.ID] = channel
		}

		pages := resp.Paging.Pages
		if pages <= page {
			break
		}
		if page == maxPages {
			partial = true
		}
	}

	channels := make([]Channel, 0, len(byID))
	for _, channel := range byID {
		channels = append(channels, channel)
	}
	sort.Slice(channels, func(i, j int) bool {
		if channels[i].ActivityCount != channels[j].ActivityCount {
			return channels[i].ActivityCount > channels[j].ActivityCount
		}
		if !channels[i].LastActivityAt.Equal(channels[j].LastActivityAt) {
			return channels[i].LastActivityAt.After(channels[j].LastActivityAt)
		}
		return channels[i].ID < channels[j].ID
	})

	return Snapshot{
		SchemaVersion: SchemaVersion,
		Workspace:     Workspace{ID: auth.TeamID},
		User:          User{ID: auth.UserID},
		Window:        Window{Start: start, End: now, Days: windowDays},
		RefreshedAt:   now,
		Partial:       partial,
		Channels:      channels,
	}, nil
}

func parseSlackTimestamp(raw string) (time.Time, error) {
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || len(parts) == 0 || parts[0] == "" {
		return time.Time{}, fmt.Errorf("invalid Slack timestamp %q", raw)
	}
	seconds, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid Slack timestamp %q: %w", raw, err)
	}
	nanos := int64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if fraction == "" {
			return time.Time{}, fmt.Errorf("invalid Slack timestamp %q", raw)
		}
		if len(fraction) > 9 {
			fraction = fraction[:9]
		}
		fraction += strings.Repeat("0", 9-len(fraction))
		nanos, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid Slack timestamp %q: %w", raw, err)
		}
	}
	return time.Unix(seconds, nanos).UTC(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func cachePath(base, teamID, userID string) (string, error) {
	if base == "" {
		var err error
		base, err = os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("slack context cache directory: %w", err)
		}
	}
	return filepath.Join(base, "slack-cli", "context", safeComponent(teamID), safeComponent(userID)+".json"), nil
}

func safeComponent(value string) string {
	original := value
	if value != "" {
		for _, r := range value {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				continue
			}
			value = ""
			break
		}
	}
	if value != "" {
		return value
	}
	hash := sha256.Sum256([]byte(original))
	return "id-" + hex.EncodeToString(hash[:8])
}

func readSnapshot(path string) (*Snapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat cache: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, errors.New("Slack context cache permissions are too broad")
	}
	for _, directory := range cacheDirectories(path) {
		directoryInfo, err := os.Stat(directory)
		if err != nil {
			return nil, fmt.Errorf("stat cache directory: %w", err)
		}
		if !directoryInfo.IsDir() || directoryInfo.Mode().Perm()&0o077 != 0 {
			return nil, errors.New("Slack context cache directory permissions are too broad")
		}
	}
	var snapshot Snapshot
	if err := json.NewDecoder(file).Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("decode cache: %w", err)
	}
	if snapshot.SchemaVersion != SchemaVersion || snapshot.Workspace.ID == "" || snapshot.User.ID == "" || snapshot.RefreshedAt.IsZero() {
		return nil, errors.New("invalid Slack context cache schema")
	}
	return &snapshot, nil
}

func writeSnapshot(path string, snapshot Snapshot) error {
	if err := ensureCacheDirectories(path); err != nil {
		return err
	}
	directory := filepath.Dir(path)
	tmp, err := os.CreateTemp(directory, ".context-*.tmp")
	if err != nil {
		return fmt.Errorf("create cache temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(cacheFileMode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("protect cache file: %w", err)
	}
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(snapshot); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("encode cache: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close cache temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace cache: %w", err)
	}
	return nil
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, cacheDirectoryMode); err != nil {
		return fmt.Errorf("create Slack context cache directory: %w", err)
	}
	if err := os.Chmod(path, cacheDirectoryMode); err != nil {
		return fmt.Errorf("protect Slack context cache directory: %w", err)
	}
	return nil
}

func cacheDirectories(path string) []string {
	teamDirectory := filepath.Dir(path)
	contextDirectory := filepath.Dir(teamDirectory)
	cacheRoot := filepath.Dir(contextDirectory)
	return []string{cacheRoot, contextDirectory, teamDirectory}
}

func ensureCacheDirectories(path string) error {
	for _, directory := range cacheDirectories(path) {
		if err := ensurePrivateDirectory(directory); err != nil {
			return err
		}
	}
	return nil
}

var errLockLost = errors.New("slack context cache lock lost")

func withLock(ctx context.Context, path string, fn func(context.Context) error) error {
	return withLockInterval(ctx, path, lockHeartbeatEvery, fn)
}

func withLockInterval(ctx context.Context, path string, heartbeatEvery time.Duration, fn func(context.Context) error) error {
	owner := fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano())
	for attempt := 0; attempt < lockWaitAttempts; attempt++ {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, cacheFileMode)
		if err == nil {
			if _, writeErr := file.WriteString(owner); writeErr != nil {
				_ = file.Close()
				_ = os.Remove(path)
				return fmt.Errorf("write cache lock: %w", writeErr)
			}
			if closeErr := file.Close(); closeErr != nil {
				_ = os.Remove(path)
				return fmt.Errorf("close cache lock: %w", closeErr)
			}
			lockCtx, cancel := context.WithCancel(ctx)
			stopHeartbeat := make(chan struct{})
			heartbeatDone := make(chan struct{})
			lockLost := make(chan struct{})
			go maintainLockWithInterval(path, owner, stopHeartbeat, heartbeatDone, lockLost, cancel, heartbeatEvery)
			defer func() {
				close(stopHeartbeat)
				<-heartbeatDone
				releaseLock(path, owner)
				cancel()
			}()
			fnErr := fn(lockCtx)
			select {
			case <-lockLost:
				return errLockLost
			default:
				return fnErr
			}
		}
		if !os.IsExist(err) {
			return fmt.Errorf("create cache lock: %w", err)
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > lockStaleAfter {
			if removeErr := os.Remove(path); removeErr == nil {
				continue
			}
		}
		timer := time.NewTimer(lockWaitInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	return errors.New("slack context cache is busy")
}

func releaseLock(path, owner string) {
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != owner {
		return
	}
	_ = os.Remove(path)
}

func maintainLock(path, owner string, stop <-chan struct{}, done chan<- struct{}, lost chan<- struct{}, cancel context.CancelFunc) {
	maintainLockWithInterval(path, owner, stop, done, lost, cancel, lockHeartbeatEvery)
}

func maintainLockWithInterval(path, owner string, stop <-chan struct{}, done chan<- struct{}, lost chan<- struct{}, cancel context.CancelFunc, heartbeatEvery time.Duration) {
	defer close(done)
	ticker := time.NewTicker(heartbeatEvery)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			if !refreshLock(path, owner, now) {
				close(lost)
				cancel()
				return
			}
		}
	}
}

func refreshLock(path, owner string, now time.Time) bool {
	contents, err := os.ReadFile(path)
	if err != nil || string(contents) != owner {
		return false
	}
	return os.Chtimes(path, now, now) == nil
}

// ErrorKind intentionally returns a coarse category so stale-cache diagnostics
// do not echo arbitrary API error text into normal command output.
func ErrorKind(err error) string {
	if err == nil {
		return ""
	}
	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "cache lock lost"):
		return "lock_lost"
	case strings.Contains(lower, "missing_scope"):
		return "missing_scope"
	case strings.Contains(lower, "ratelimited"), strings.Contains(lower, "rate limit"):
		return "ratelimited"
	case strings.Contains(lower, "timeout"), strings.Contains(lower, "network"), strings.Contains(lower, "connection"):
		return "network"
	case strings.Contains(lower, "token"), strings.Contains(lower, "auth"):
		return "authentication"
	default:
		return "api"
	}
}
