package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"

	"github.com/koinunopochi/slack-cli/internal/client"
	"github.com/koinunopochi/slack-cli/internal/config"
	"github.com/koinunopochi/slack-cli/internal/errs"
	"github.com/koinunopochi/slack-cli/internal/output"
)

var (
	searchFilesCount         int
	searchFilesPage          int
	searchFilesSort          string
	searchFilesSortDirection string
	searchFilesHighlight     bool
	searchFilesTeamID        string
)

var searchFilesCmd = &cobra.Command{
	Use:   "search-files <query>",
	Short: "Search Slack files (User Token only)",
	Long: `Search files across Slack using the search.files API. Supports the same query
syntax as search.messages (from:, in:, after:, before:, "phrase", has:link, ...).

User Token only — requires the legacy search:read scope, which Bot Tokens
cannot obtain.

Examples:
  slack search-files "design from:@okayama after:2026-05-01"
  slack search-files "schema.png in:#team-dev-ict-develop"`,
	Args: cobra.ExactArgs(1),
	RunE: runSearchFiles,
}

func init() {
	searchFilesCmd.Flags().IntVar(&searchFilesCount, "count", 20, "hits per page (max 100)")
	searchFilesCmd.Flags().IntVar(&searchFilesPage, "page", 1, "page number (1-based)")
	searchFilesCmd.Flags().StringVar(&searchFilesSort, "sort", "score", `"score" or "timestamp"`)
	searchFilesCmd.Flags().StringVar(&searchFilesSortDirection, "sort-direction", "desc", `"asc" or "desc"`)
	searchFilesCmd.Flags().BoolVar(&searchFilesHighlight, "highlight", false, "include highlight markers")
	searchFilesCmd.Flags().StringVar(&searchFilesTeamID, "team-id", "", "Enterprise Grid: filter by team ID")
	RootCmd.AddCommand(searchFilesCmd)
}

func runSearchFiles(c *cobra.Command, args []string) error {
	tt, err := config.ParseTokenType(FlagTokenType)
	if err != nil {
		return err
	}
	if tt != config.TokenTypeUser {
		return fmt.Errorf("search-files requires --token-type user (search.files is User Token only)")
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

	params := slack.NewSearchParameters()
	params.Count = searchFilesCount
	params.Page = searchFilesPage
	params.Sort = searchFilesSort
	params.SortDirection = searchFilesSortDirection
	params.Highlight = searchFilesHighlight
	if searchFilesTeamID != "" {
		params.TeamID = searchFilesTeamID
	}

	resp, err := cl.SearchFilesContext(ctx, args[0], params)
	if err != nil {
		return errs.Enrich(err, []string{"search:read", "files:read"})
	}

	var nextPage any
	if resp.Paging.Page < resp.Paging.Pages {
		nextPage = resp.Paging.Page + 1
	}

	out := map[string]any{
		"query":      args[0],
		"matches":    resp.Matches,
		"total":      resp.Total,
		"page":       resp.Paging.Page,
		"page_count": resp.Paging.Pages,
		"next_page":  nextPage,
	}
	return output.Emit(out, fmtt, FlagOut)
}
