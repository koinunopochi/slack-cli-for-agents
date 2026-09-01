package contextcache

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/slack-go/slack"
)

type fakeClient struct {
	auth      *slack.AuthTestResponse
	responses map[int]*slack.SearchMessages
	searchErr error
	calls     []slack.SearchParameters
	queries   []string
}

func (f *fakeClient) AuthTestContext(context.Context) (*slack.AuthTestResponse, error) {
	return f.auth, nil
}

func (f *fakeClient) SearchMessagesContext(_ context.Context, query string, params slack.SearchParameters) (*slack.SearchMessages, error) {
	f.calls = append(f.calls, params)
	f.queries = append(f.queries, query)
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	if response, ok := f.responses[params.Page]; ok {
		return response, nil
	}
	return &slack.SearchMessages{Paging: slack.Paging{Page: params.Page, Pages: params.Page}}, nil
}

func slackTS(t time.Time) string {
	return fmt.Sprintf("%d.%06d", t.Unix(), t.Nanosecond()/1000)
}

func TestEnsureBuildsAndRefreshesRollingContextWithoutMessageBodies(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	client := &fakeClient{
		auth: &slack.AuthTestResponse{TeamID: "T1", UserID: "U1"},
		responses: map[int]*slack.SearchMessages{
			1: {
				Matches: []slack.SearchMessage{
					{User: "U1", Timestamp: slackTS(now.Add(-time.Hour)), Text: "do not cache", Channel: slack.CtxChannel{ID: "C1", Name: "alpha"}},
					{User: "U1", Timestamp: slackTS(now.Add(-2 * time.Hour)), Channel: slack.CtxChannel{ID: "C1", Name: "alpha"}},
					{User: "U1", Timestamp: slackTS(now.Add(-3 * time.Hour)), Channel: slack.CtxChannel{ID: "C2", Name: "beta", IsPrivate: true}},
					{Timestamp: slackTS(now.Add(-4 * time.Hour)), Channel: slack.CtxChannel{ID: "C3", Name: "other-user"}},
					{User: "U2", Timestamp: slackTS(now.Add(-5 * time.Hour)), Channel: slack.CtxChannel{ID: "C4", Name: "another-user"}},
				},
				Paging: slack.Paging{Page: 1, Pages: 1},
			},
		},
	}
	cacheDir := t.TempDir()
	result, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fresh" || !result.Refreshed {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Snapshot.Channels) != 2 || result.Snapshot.Channels[0].ID != "C1" || result.Snapshot.Channels[0].ActivityCount != 2 {
		t.Fatalf("unexpected channels: %+v", result.Snapshot.Channels)
	}
	if len(client.queries) != 1 || client.queries[0] != "from:<@U1> after:2026-08-01" {
		t.Fatalf("unexpected search query: %v", client.queries)
	}
	cacheFile := filepath.Join(cacheDir, "slack-cli", "context", "T1", "U1.json")
	body, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(body, []byte("\"text\"")) || bytes.Contains(body, []byte("do not cache")) {
		t.Fatalf("cache unexpectedly contains message text: %s", body)
	}
	info, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != cacheFileMode {
		t.Fatalf("cache mode = %o, want %o", info.Mode().Perm(), cacheFileMode)
	}
	directoryInfo, err := os.Stat(filepath.Dir(cacheFile))
	if err != nil {
		t.Fatal(err)
	}
	if directoryInfo.Mode().Perm() != cacheDirectoryMode {
		t.Fatalf("cache directory mode = %o, want %o", directoryInfo.Mode().Perm(), cacheDirectoryMode)
	}
	if err := os.Chmod(filepath.Dir(cacheFile), 0o755); err != nil {
		t.Fatal(err)
	}
	client.calls = nil
	if result, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now.Add(time.Hour) }}); err != nil {
		t.Fatal(err)
	} else if result.Status != StatusFresh || result.Refreshed || len(client.calls) != 0 {
		t.Fatalf("broad-permission cache was not safely reused: %+v calls=%d", result, len(client.calls))
	}
	directoryInfo, err = os.Stat(filepath.Dir(cacheFile))
	if err != nil {
		t.Fatal(err)
	}
	if directoryInfo.Mode().Perm() != cacheDirectoryMode {
		t.Fatalf("broad-permission cache directory mode = %o, want %o", directoryInfo.Mode().Perm(), cacheDirectoryMode)
	}
	for _, directory := range cacheDirectories(cacheFile) {
		info, err := os.Stat(directory)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != cacheDirectoryMode {
			t.Fatalf("cache directory %s mode = %o, want %o", directory, info.Mode().Perm(), cacheDirectoryMode)
		}
	}
	if err := os.Chmod(filepath.Dir(cacheFile), cacheDirectoryMode); err != nil {
		t.Fatal(err)
	}

	client.responses[1] = &slack.SearchMessages{
		Matches: []slack.SearchMessage{
			{User: "U1", Timestamp: slackTS(now.Add(24 * time.Hour)), Channel: slack.CtxChannel{ID: "C3", Name: "new"}},
		},
		Paging: slack.Paging{Page: 1, Pages: 1},
	}
	second, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now.Add(25 * time.Hour) }})
	if err != nil {
		t.Fatal(err)
	}
	if !second.Refreshed || len(second.Snapshot.Channels) != 1 || second.Snapshot.Channels[0].ID != "C3" {
		t.Fatalf("rolling refresh did not replace old window: %+v", second)
	}
}

func TestEnsureRejectsCacheForAnotherIdentity(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	cacheDir := t.TempDir()
	client := &fakeClient{
		auth:      &slack.AuthTestResponse{TeamID: "T1", UserID: "U1"},
		responses: map[int]*slack.SearchMessages{1: {Paging: slack.Paging{Page: 1, Pages: 1}}},
	}
	if _, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now }}); err != nil {
		t.Fatal(err)
	}
	cacheFile := filepath.Join(cacheDir, "slack-cli", "context", "T1", "U1.json")
	if err := writeSnapshot(cacheFile, Snapshot{
		SchemaVersion: SchemaVersion,
		Workspace:     Workspace{ID: "T2"},
		User:          User{ID: "U2"},
		Window:        Window{Days: DefaultWindowDays},
		RefreshedAt:   now,
	}); err != nil {
		t.Fatal(err)
	}
	client.calls = nil
	result, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now.Add(time.Hour) }})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Refreshed || result.Snapshot.Workspace.ID != "T1" || result.Snapshot.User.ID != "U1" || len(client.calls) != 1 {
		t.Fatalf("cache for another identity was reused: %+v calls=%d", result, len(client.calls))
	}
}

func TestWithLockDoesNotRemoveReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "context.lock")
	err := withLock(context.Background(), path, func(context.Context) error {
		if err := os.Remove(path); err != nil {
			return err
		}
		return os.WriteFile(path, []byte("replacement"), cacheFileMode)
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "replacement" {
		t.Fatalf("replacement lock = %q", body)
	}
}

func TestWithLockCancelsRefreshWhenLockIsLost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "context.lock")
	started := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- withLockInterval(context.Background(), path, time.Millisecond, func(lockCtx context.Context) error {
			close(started)
			<-lockCtx.Done()
			return lockCtx.Err()
		})
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("lock callback did not start")
	}
	if err := os.WriteFile(path, []byte("replacement"), cacheFileMode); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if !errors.Is(err, errLockLost) {
			t.Fatalf("lock loss error = %v, want %v", err, errLockLost)
		}
	case <-time.After(time.Second):
		t.Fatal("lock loss did not cancel refresh")
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "replacement" {
		t.Fatalf("replacement lock = %q", body)
	}
}

func TestEnsureUsesFreshCacheWithoutSearch(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	cacheDir := t.TempDir()
	client := &fakeClient{
		auth:      &slack.AuthTestResponse{TeamID: "T1", UserID: "U1"},
		responses: map[int]*slack.SearchMessages{1: {Paging: slack.Paging{Page: 1, Pages: 1}}},
	}
	if _, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now }}); err != nil {
		t.Fatal(err)
	}
	client.calls = nil
	result, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now.Add(time.Hour) }})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "fresh" || result.Refreshed || len(client.calls) != 0 {
		t.Fatalf("fresh cache was not reused: %+v calls=%d", result, len(client.calls))
	}
}

func TestEnsureRebuildsCacheWithBroadPermissions(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	cacheDir := t.TempDir()
	client := &fakeClient{
		auth:      &slack.AuthTestResponse{TeamID: "T1", UserID: "U1"},
		responses: map[int]*slack.SearchMessages{1: {Paging: slack.Paging{Page: 1, Pages: 1}}},
	}
	if _, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now }}); err != nil {
		t.Fatal(err)
	}
	cacheFile := filepath.Join(cacheDir, "slack-cli", "context", "T1", "U1.json")
	if err := os.Chmod(cacheFile, 0o644); err != nil {
		t.Fatal(err)
	}
	client.calls = nil
	result, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now.Add(time.Hour) }})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Refreshed || len(client.calls) != 1 {
		t.Fatalf("broad-permission cache was reused: %+v calls=%d", result, len(client.calls))
	}
}

func TestEnsureReturnsStaleCacheWhenRefreshFails(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	cacheDir := t.TempDir()
	client := &fakeClient{
		auth:      &slack.AuthTestResponse{TeamID: "T1", UserID: "U1"},
		responses: map[int]*slack.SearchMessages{1: {Paging: slack.Paging{Page: 1, Pages: 1}}},
	}
	if _, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now }}); err != nil {
		t.Fatal(err)
	}
	client.searchErr = errors.New("missing_scope")
	result, err := Ensure(context.Background(), client, Options{CacheDir: cacheDir, Now: func() time.Time { return now.Add(25 * time.Hour) }})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "stale" || result.RefreshError != "missing_scope" {
		t.Fatalf("unexpected stale result: %+v", result)
	}
}

func TestEnsureMarksTruncatedSearch(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	client := &fakeClient{
		auth:      &slack.AuthTestResponse{TeamID: "T1", UserID: "U1"},
		responses: map[int]*slack.SearchMessages{1: {Paging: slack.Paging{Page: 1, Pages: 2}}},
	}
	result, err := Ensure(context.Background(), client, Options{CacheDir: t.TempDir(), MaxSearchPages: 1, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != StatusPartial || !result.Snapshot.Partial {
		t.Fatal("expected partial cache when the search page cap is reached")
	}
}

func TestParseSlackTimestamp(t *testing.T) {
	got, err := parseSlackTimestamp("1716000000.123456")
	if err != nil {
		t.Fatal(err)
	}
	if got.Unix() != 1716000000 || got.Nanosecond() != 123456000 {
		t.Fatalf("timestamp = %s", got)
	}
	if _, err := parseSlackTimestamp("not-a-timestamp"); err == nil {
		t.Fatal("expected malformed timestamp to fail")
	}
}

func TestErrorKindDoesNotEchoDetails(t *testing.T) {
	if got := ErrorKind(errors.New("https://slack.example/?token=secret network timeout")); got != "network" {
		t.Fatalf("ErrorKind = %q", got)
	}
}
