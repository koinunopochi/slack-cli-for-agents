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
	searchChannelsTypes           string
	searchChannelsLimit           int
	searchChannelsCursor          string
	searchChannelsExcludeArchived bool
	searchChannelsIncludeArchived bool
	searchChannelsQuery           string
	searchChannelsTeamID          string
	searchChannelsMaxPages        int
	searchChannelsAll             bool
)

var searchChannelsCmd = &cobra.Command{
	Use:   "search-channels",
	Short: "List / filter Slack channels (conversations.list)",
	Long: `List Slack conversations (public/private channels, MPIM, IM) via conversations.list.
Slack's API has no server-side query; --query applies a local case-insensitive substring
match on channel name after fetch.

Pagination
----------
Because --query filters locally, the first --limit channels may not contain
any matches in a large workspace. Two ways to walk further:

  --max-pages N   Fetch up to N pages (default 1). Walks the next_cursor chain
                  automatically until either N pages are fetched or the chain
                  ends.
  --all           Walk every page until next_cursor is empty, ignoring
                  --max-pages. Use this when you want absolutely every channel.

The output includes pages_fetched so callers can see how far the walk got.`,
	Args: cobra.NoArgs,
	RunE: runSearchChannels,
}

func init() {
	searchChannelsCmd.Flags().StringVar(&searchChannelsTypes, "types", "public_channel",
		"comma-separated: public_channel,private_channel,mpim,im")
	searchChannelsCmd.Flags().IntVar(&searchChannelsLimit, "limit", 200,
		"channels per request (max 999)")
	searchChannelsCmd.Flags().StringVar(&searchChannelsCursor, "cursor", "", "pagination cursor (start from)")
	searchChannelsCmd.Flags().BoolVar(&searchChannelsExcludeArchived, "exclude-archived", true, "exclude archived channels")
	searchChannelsCmd.Flags().BoolVar(&searchChannelsIncludeArchived, "include-archived", false, "include archived (overrides --exclude-archived)")
	searchChannelsCmd.Flags().StringVar(&searchChannelsQuery, "query", "", "substring match on channel name (case-insensitive, applied locally)")
	searchChannelsCmd.Flags().StringVar(&searchChannelsTeamID, "team-id", "", "Enterprise Grid: workspace filter")
	searchChannelsCmd.Flags().IntVar(&searchChannelsMaxPages, "max-pages", 1,
		"fetch up to this many pages (walks next_cursor automatically). "+
			"Raise this when --query rarely matches in the first page.")
	searchChannelsCmd.Flags().BoolVar(&searchChannelsAll, "all", false,
		"walk every page until next_cursor is empty; ignores --max-pages")
	RootCmd.AddCommand(searchChannelsCmd)
}

func runSearchChannels(c *cobra.Command, args []string) error {
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

	types := splitCSV(searchChannelsTypes)
	excludeArchived := searchChannelsExcludeArchived && !searchChannelsIncludeArchived

	var (
		all          []slack.Channel
		cursor       = searchChannelsCursor
		pagesFetched int
		nextCursor   string
	)
	for {
		params := &slack.GetConversationsParameters{
			Cursor:          cursor,
			ExcludeArchived: excludeArchived,
			Limit:           searchChannelsLimit,
			Types:           types,
			TeamID:          searchChannelsTeamID,
		}
		channels, nc, err := cl.GetConversationsContext(ctx, params)
		if err != nil {
			return errs.Enrich(err, neededScopesForChannelTypes(types))
		}
		all = append(all, channels...)
		pagesFetched++
		nextCursor = nc

		// Stop when there is nothing left.
		if nc == "" {
			break
		}
		// --all walks until next_cursor is empty, ignoring --max-pages.
		// Otherwise --max-pages alone decides how far we walk: --max-pages N
		// means "fetch up to N pages", which is the obvious reading.
		// --max-pages 0 with --all means "no cap".
		if !searchChannelsAll && pagesFetched >= searchChannelsMaxPages {
			break
		}
		cursor = nc
	}

	if q := strings.ToLower(strings.TrimSpace(searchChannelsQuery)); q != "" {
		filtered := all[:0]
		for _, ch := range all {
			if strings.Contains(strings.ToLower(ch.Name), q) ||
				strings.Contains(strings.ToLower(ch.NameNormalized), q) {
				filtered = append(filtered, ch)
			}
		}
		all = filtered
	}

	out := map[string]any{
		"channels":      all,
		"next_cursor":   nextCursor,
		"query":         searchChannelsQuery,
		"pages_fetched": pagesFetched,
	}
	return output.Emit(out, fmtt, FlagOut)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// neededScopesForChannelTypes maps the --types selection to the conversations.list
// scopes Slack will check. We over-report rather than under-report: showing too
// many candidate scopes in a missing_scope diagnostic is fine, the user will
// recognize the missing one. Hiding the right one is what hurts.
func neededScopesForChannelTypes(types []string) []string {
	if len(types) == 0 {
		return []string{"channels:read"}
	}
	scopes := make([]string, 0, len(types))
	for _, t := range types {
		switch t {
		case "public_channel":
			scopes = append(scopes, "channels:read")
		case "private_channel":
			scopes = append(scopes, "groups:read")
		case "im":
			scopes = append(scopes, "im:read")
		case "mpim":
			scopes = append(scopes, "mpim:read")
		}
	}
	return scopes
}
