package cmd

import (
	"context"
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
	searchUsersLimit          int
	searchUsersCursor         string
	searchUsersQuery          string
	searchUsersMaxPages       int
	searchUsersIncludeDeleted bool
	searchUsersTeamID         string
)

var searchUsersCmd = &cobra.Command{
	Use:   "search-users",
	Short: "List / filter Slack users (users.list)",
	Long: `List Slack users via users.list (GetUsersPaginated) and optionally filter
locally by substring against name / real_name / display_name.

Large workspaces can return many pages; use --max-pages to bound the fetch
and re-run with --cursor <next_cursor> from the previous output to paginate.`,
	Args: cobra.NoArgs,
	RunE: runSearchUsers,
}

func init() {
	searchUsersCmd.Flags().IntVar(&searchUsersLimit, "limit", 200, "users per request (recommended max 200)")
	searchUsersCmd.Flags().StringVar(&searchUsersCursor, "cursor", "", "pagination cursor (start from)")
	searchUsersCmd.Flags().StringVar(&searchUsersQuery, "query", "", "substring match on name / real_name / display_name (case-insensitive, local)")
	searchUsersCmd.Flags().IntVar(&searchUsersMaxPages, "max-pages", 1, "stop after fetching this many pages")
	searchUsersCmd.Flags().BoolVar(&searchUsersIncludeDeleted, "include-deleted", false, "include deleted/deactivated users")
	searchUsersCmd.Flags().StringVar(&searchUsersTeamID, "team-id", "", "Enterprise Grid: org-level token uses this")
	RootCmd.AddCommand(searchUsersCmd)
}

func runSearchUsers(c *cobra.Command, args []string) error {
	tt, err := config.ParseTokenType(FlagTokenType)
	if err != nil {
		return err
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

	opts := []slack.GetUsersOption{
		slack.GetUsersOptionLimit(searchUsersLimit),
	}
	if searchUsersCursor != "" {
		opts = append(opts, slack.GetUsersOptionCursor(searchUsersCursor))
	}
	if searchUsersTeamID != "" {
		opts = append(opts, slack.GetUsersOptionTeamID(searchUsersTeamID))
	}

	p := cl.GetUsersPaginated(opts...)
	var all []slack.User
	pages := 0
	var loopErr error
	for {
		p, loopErr = p.Next(ctx)
		if p.Done(loopErr) {
			break
		}
		if e := p.Failure(loopErr); e != nil {
			return errs.Enrich(e, []string{"users:read"})
		}
		all = append(all, p.Users...)
		pages++
		if pages >= searchUsersMaxPages {
			break
		}
	}
	if e := p.Failure(loopErr); e != nil {
		return errs.Enrich(e, []string{"users:read"})
	}

	if !searchUsersIncludeDeleted {
		out := all[:0]
		for _, u := range all {
			if !u.Deleted {
				out = append(out, u)
			}
		}
		all = out
	}

	if q := strings.ToLower(strings.TrimSpace(searchUsersQuery)); q != "" {
		out := all[:0]
		for _, u := range all {
			if strings.Contains(strings.ToLower(u.Name), q) ||
				strings.Contains(strings.ToLower(u.RealName), q) ||
				strings.Contains(strings.ToLower(u.Profile.DisplayName), q) {
				out = append(out, u)
			}
		}
		all = out
	}

	nextCursor := ""
	if pages >= searchUsersMaxPages && !p.Done(nil) {
		nextCursor = p.Cursor
	}

	res := map[string]any{
		"users":       all,
		"next_cursor": nextCursor,
		"page_count":  pages,
		"query":       searchUsersQuery,
	}
	return output.Emit(res, fmtt, FlagOut)
}
