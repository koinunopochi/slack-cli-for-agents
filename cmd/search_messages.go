package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"

	"github.com/koinunopochi/slack-cli/internal/client"
	"github.com/koinunopochi/slack-cli/internal/config"
	"github.com/koinunopochi/slack-cli/internal/output"
)

var (
	searchMessagesCount         int
	searchMessagesPage          int
	searchMessagesSort          string
	searchMessagesSortDirection string
	searchMessagesHighlight     bool
	searchMessagesTeamID        string
)

var searchMessagesCmd = &cobra.Command{
	Use:   "search-messages <query>",
	Short: "Search Slack messages (User Token only)",
	Long: `Search Slack messages across the workspace using Slack's search.messages API.
Supports Slack query syntax (from:, in:, before:, after:, has:, "phrase"...).
User Token only: the underlying API requires the legacy search:read scope,
which is not available to Bot Tokens. Pass --token-type user.`,
	Args: cobra.ExactArgs(1),
	RunE: runSearchMessages,
}

func init() {
	searchMessagesCmd.Flags().IntVar(&searchMessagesCount, "count", 20, "hits per page (max 100)")
	searchMessagesCmd.Flags().IntVar(&searchMessagesPage, "page", 1, "page number (1-based, max 100)")
	searchMessagesCmd.Flags().StringVar(&searchMessagesSort, "sort", "score", "sort by \"score\" or \"timestamp\"")
	searchMessagesCmd.Flags().StringVar(&searchMessagesSortDirection, "sort-direction", "desc", "\"asc\" or \"desc\"")
	searchMessagesCmd.Flags().BoolVar(&searchMessagesHighlight, "highlight", false, "include highlight markers")
	searchMessagesCmd.Flags().StringVar(&searchMessagesTeamID, "team-id", "", "filter by team ID (Enterprise Grid)")
	RootCmd.AddCommand(searchMessagesCmd)
}

func runSearchMessages(c *cobra.Command, args []string) error {
	tt, err := config.ParseTokenType(FlagTokenType)
	if err != nil {
		return err
	}

	if tt != config.TokenTypeUser {
		return fmt.Errorf("search-messages requires --token-type user (Slack API limitation: search.messages is User Token only)")
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
	params.Count = searchMessagesCount
	params.Page = searchMessagesPage
	params.Sort = searchMessagesSort
	params.SortDirection = searchMessagesSortDirection
	params.Highlight = searchMessagesHighlight
	if searchMessagesTeamID != "" {
		params.TeamID = searchMessagesTeamID
	}

	resp, err := cl.SearchMessagesContext(ctx, args[0], params)
	if err != nil {
		return err
	}

	nextPage := any(nil)
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
