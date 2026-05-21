package cmd

import (
	"context"
	"strings"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"

	"github.com/koinunopochi/slack-cli/internal/client"
	"github.com/koinunopochi/slack-cli/internal/config"
	"github.com/koinunopochi/slack-cli/internal/output"
)

var (
	searchChannelsTypes           string
	searchChannelsLimit           int
	searchChannelsCursor          string
	searchChannelsExcludeArchived bool
	searchChannelsIncludeArchived bool
	searchChannelsQuery           string
	searchChannelsTeamID          string
)

var searchChannelsCmd = &cobra.Command{
	Use:   "search-channels",
	Short: "List / filter Slack channels (conversations.list)",
	Long: `List Slack conversations (public/private channels, MPIM, IM) via conversations.list.
Slack's API has no server-side query; --query applies a local case-insensitive substring
match on channel name after fetch. Use --limit large and --cursor for pagination.`,
	Args: cobra.NoArgs,
	RunE: runSearchChannels,
}

func init() {
	searchChannelsCmd.Flags().StringVar(&searchChannelsTypes, "types", "public_channel",
		"comma-separated: public_channel,private_channel,mpim,im")
	searchChannelsCmd.Flags().IntVar(&searchChannelsLimit, "limit", 200,
		"channels per request (max 999)")
	searchChannelsCmd.Flags().StringVar(&searchChannelsCursor, "cursor", "", "pagination cursor")
	searchChannelsCmd.Flags().BoolVar(&searchChannelsExcludeArchived, "exclude-archived", true, "exclude archived channels")
	searchChannelsCmd.Flags().BoolVar(&searchChannelsIncludeArchived, "include-archived", false, "include archived (overrides --exclude-archived)")
	searchChannelsCmd.Flags().StringVar(&searchChannelsQuery, "query", "", "substring match on channel name (case-insensitive, applied locally)")
	searchChannelsCmd.Flags().StringVar(&searchChannelsTeamID, "team-id", "", "Enterprise Grid: workspace filter")
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

	params := &slack.GetConversationsParameters{
		Cursor:          searchChannelsCursor,
		ExcludeArchived: excludeArchived,
		Limit:           searchChannelsLimit,
		Types:           types,
		TeamID:          searchChannelsTeamID,
	}
	channels, nextCursor, err := cl.GetConversationsContext(ctx, params)
	if err != nil {
		return err
	}

	if q := strings.ToLower(strings.TrimSpace(searchChannelsQuery)); q != "" {
		filtered := channels[:0]
		for _, ch := range channels {
			if strings.Contains(strings.ToLower(ch.Name), q) ||
				strings.Contains(strings.ToLower(ch.NameNormalized), q) {
				filtered = append(filtered, ch)
			}
		}
		channels = filtered
	}

	out := map[string]any{
		"channels":    channels,
		"next_cursor": nextCursor,
		"query":       searchChannelsQuery,
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
