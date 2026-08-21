package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"

	"github.com/koinunopochi/slack-cli-for-agents/internal/client"
	"github.com/koinunopochi/slack-cli-for-agents/internal/config"
	"github.com/koinunopochi/slack-cli-for-agents/internal/errs"
	"github.com/koinunopochi/slack-cli-for-agents/internal/output"
)

var (
	userActivityDays          int
	userActivityCount         int
	userActivityPage          int
	userActivityIn            string
	userActivitySort          string
	userActivitySortDirection string
	userActivityMaxUserPages  int
)

var userActivityCmd = &cobra.Command{
	Use:   "user-activity <name-or-id>",
	Short: "Recent messages from a Slack user (resolve name -> id and search.messages)",
	Long: `Resolve a Slack user (by ID U..., or by substring match on name / real_name /
display_name via users.list) and fetch their recent messages via search.messages.

User Token only — search.messages requires the legacy search:read scope which
is not available to Bot Tokens.

Examples:
  slack user-activity U0123456789 --days 7
  slack user-activity alice --days 14 --in '#team-example'`,
	Args: cobra.ExactArgs(1),
	RunE: runUserActivity,
}

func init() {
	userActivityCmd.Flags().IntVar(&userActivityDays, "days", 14, "fetch messages from the last N days (Slack 'after:' clause)")
	userActivityCmd.Flags().IntVar(&userActivityCount, "count", 50, "hits per page (max 100)")
	userActivityCmd.Flags().IntVar(&userActivityPage, "page", 1, "page number (1-based, max 100)")
	userActivityCmd.Flags().StringVar(&userActivityIn, "in", "", "restrict to a single channel (added to query as in:#<name>)")
	userActivityCmd.Flags().StringVar(&userActivitySort, "sort", "timestamp", `"score" or "timestamp"`)
	userActivityCmd.Flags().StringVar(&userActivitySortDirection, "sort-direction", "desc", `"asc" or "desc"`)
	userActivityCmd.Flags().IntVar(&userActivityMaxUserPages, "max-user-pages", 30,
		"max pages to scan when resolving the user (users.list, 200 users/page). "+
			"Mid-to-large workspaces often need 20+ to surface a given name; "+
			"prefer passing a Slack user ID (Uxxxx) to skip resolution entirely.")
	RootCmd.AddCommand(userActivityCmd)
}

func runUserActivity(c *cobra.Command, args []string) error {
	tt, err := config.ParseTokenType(FlagTokenType)
	if err != nil {
		return err
	}
	if tt != config.TokenTypeUser {
		return fmt.Errorf("user-activity requires --token-type user (search.messages is User Token only)")
	}

	token, err := config.LoadToken(tt)
	if err != nil {
		return err
	}

	fmtt, err := output.ParseFormat(FlagFormat)
	if err != nil {
		return err
	}

	cl := client.New(token, client.Options{Timeout: FlagTimeout, Debug: FlagDebug})

	ctx, cancel := context.WithTimeout(c.Context(), FlagTimeout+5*time.Second)
	defer cancel()

	user, err := resolveUser(ctx, cl, args[0], userActivityMaxUserPages)
	if err != nil {
		return err
	}

	after := time.Now().Add(-time.Duration(userActivityDays) * 24 * time.Hour).Format("2006-01-02")
	parts := []string{
		fmt.Sprintf("from:@%s", user.Name),
		fmt.Sprintf("after:%s", after),
	}
	if userActivityIn != "" {
		parts = append(parts, fmt.Sprintf("in:%s", normalizeIn(userActivityIn)))
	}
	query := strings.Join(parts, " ")

	params := slack.NewSearchParameters()
	params.Count = userActivityCount
	params.Page = userActivityPage
	params.Sort = userActivitySort
	params.SortDirection = userActivitySortDirection

	resp, err := cl.SearchMessagesContext(ctx, query, params)
	if err != nil {
		return errs.Enrich(err, []string{"search:read", "users:read"})
	}

	var nextPage any
	if resp.Paging.Page < resp.Paging.Pages {
		nextPage = resp.Paging.Page + 1
	}

	out := map[string]any{
		"user": map[string]any{
			"id":           user.ID,
			"name":         user.Name,
			"real_name":    user.RealName,
			"display_name": user.Profile.DisplayName,
		},
		"query":      query,
		"matches":    resp.Matches,
		"total":      resp.Total,
		"page":       resp.Paging.Page,
		"page_count": resp.Paging.Pages,
		"next_page":  nextPage,
	}
	return output.Emit(out, fmtt, FlagOut)
}

// resolveUser returns a slack.User matching the input. If the input looks
// like a Slack ID (e.g. U0123456789) users.info is called directly.
// Otherwise users.list is paginated and the best substring match against
// name / real_name / display_name is used. Exact matches on name or display
// name win over substring matches; multiple matches at the same precedence
// produce an "ambiguous user" error listing the candidates.
func resolveUser(ctx context.Context, cl *slack.Client, input string, maxPages int) (*slack.User, error) {
	if looksLikeUserID(input) {
		u, err := cl.GetUserInfoContext(ctx, input)
		if err != nil {
			return nil, errs.Enrich(fmt.Errorf("users.info(%s): %w", input, err), []string{"users:read"})
		}
		return u, nil
	}

	needle := strings.ToLower(strings.TrimSpace(input))
	if needle == "" {
		return nil, fmt.Errorf("empty user identifier")
	}

	p := cl.GetUsersPaginated(slack.GetUsersOptionLimit(200))
	var candidates []slack.User
	var exact []slack.User
	pages := 0
	var loopErr error
	for {
		p, loopErr = p.Next(ctx)
		if p.Done(loopErr) {
			break
		}
		if e := p.Failure(loopErr); e != nil {
			return nil, errs.Enrich(e, []string{"users:read"})
		}
		for _, u := range p.Users {
			if u.Deleted {
				continue
			}
			name := strings.ToLower(u.Name)
			real := strings.ToLower(u.RealName)
			disp := strings.ToLower(u.Profile.DisplayName)
			switch {
			case name == needle || disp == needle:
				exact = append(exact, u)
			case strings.Contains(name, needle) || strings.Contains(real, needle) || strings.Contains(disp, needle):
				candidates = append(candidates, u)
			}
		}
		pages++
		if pages >= maxPages {
			break
		}
	}
	if e := p.Failure(loopErr); e != nil {
		return nil, e
	}

	switch {
	case len(exact) == 1:
		return &exact[0], nil
	case len(exact) > 1:
		return nil, ambiguousUserError(input, exact)
	case len(candidates) == 1:
		return &candidates[0], nil
	case len(candidates) > 1:
		return nil, ambiguousUserError(input, candidates)
	}
	return nil, fmt.Errorf(
		"no user matched %q (scanned %d page(s) of %d users each).\n"+
			"  hints:\n"+
			"  - large workspaces: raise --max-user-pages (default 30, allow up to total_users/200)\n"+
			"  - exact ID known: pass the Slack user ID directly, e.g. `slack user-activity U0123456789`\n"+
			"  - try `slack search-users --query %q --max-pages 50` to see how the name is spelled",
		input, pages, 200, input)
}

// looksLikeUserID returns true if s is plausibly a Slack user ID
// (starts with U or W, length >= 9, only uppercase letters and digits).
func looksLikeUserID(s string) bool {
	if len(s) < 9 {
		return false
	}
	if s[0] != 'U' && s[0] != 'W' {
		return false
	}
	for _, r := range s[1:] {
		if !(r >= '0' && r <= '9') && !(r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}

func ambiguousUserError(input string, us []slack.User) error {
	names := make([]string, 0, len(us))
	for _, u := range us {
		names = append(names, fmt.Sprintf("%s (%s, %s)", u.Name, u.ID, u.RealName))
	}
	return fmt.Errorf(
		"ambiguous user %q — %d candidates: %s\n"+
			"  hint: pass the Slack user ID (the Uxxxx part) directly to disambiguate, e.g. `slack user-activity %s`",
		input, len(us), strings.Join(names, ", "), us[0].ID)
}

// normalizeIn turns "channel-name" into "#channel-name"; "#channel-name" and
// "<#C123>" are left as-is. Slack search syntax expects the # prefix for
// channel filters.
func normalizeIn(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") || strings.HasPrefix(s, "<#") {
		return s
	}
	return "#" + s
}
