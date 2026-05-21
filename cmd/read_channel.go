package cmd

import (
	"context"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"

	"github.com/koinunopochi/slack-cli/internal/client"
	"github.com/koinunopochi/slack-cli/internal/config"
	"github.com/koinunopochi/slack-cli/internal/errs"
	"github.com/koinunopochi/slack-cli/internal/output"
	"github.com/koinunopochi/slack-cli/internal/permalink"
)

var (
	readChannelLimit           int
	readChannelCursor          string
	readChannelOldest          string
	readChannelLatest          string
	readChannelInclusive       bool
	readChannelIncludeMetadata bool
)

var readChannelCmd = &cobra.Command{
	Use:   "read-channel <channel-id>",
	Short: "Fetch recent messages in a Slack channel",
	Long: `Fetch recent messages in a Slack channel via conversations.history.

Supports public/private channels, DMs, and MPIMs. Requires the matching
*_history OAuth scope for the channel type (channels:history, groups:history,
im:history, or mpim:history) on the chosen token (user or bot).`,
	Args: cobra.ExactArgs(1),
	RunE: runReadChannel,
}

func init() {
	readChannelCmd.Flags().IntVar(&readChannelLimit, "limit", 100, "messages per request, max 999")
	readChannelCmd.Flags().StringVar(&readChannelCursor, "cursor", "", "pagination cursor (from previous \"next_cursor\")")
	readChannelCmd.Flags().StringVar(&readChannelOldest, "oldest", "", "Slack ts; only messages newer than this")
	readChannelCmd.Flags().StringVar(&readChannelLatest, "latest", "", "Slack ts; only messages older than this")
	readChannelCmd.Flags().BoolVar(&readChannelInclusive, "inclusive", false, "include messages exactly at --oldest / --latest")
	readChannelCmd.Flags().BoolVar(&readChannelIncludeMetadata, "include-metadata", false, "include all message metadata")
	RootCmd.AddCommand(readChannelCmd)
}

func runReadChannel(c *cobra.Command, args []string) error {
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

	params := &slack.GetConversationHistoryParameters{
		ChannelID:          args[0],
		Cursor:             readChannelCursor,
		Inclusive:          readChannelInclusive,
		Latest:             readChannelLatest,
		Limit:              readChannelLimit,
		Oldest:             readChannelOldest,
		IncludeAllMetadata: readChannelIncludeMetadata,
	}
	resp, err := cl.GetConversationHistoryContext(ctx, params)
	if err != nil {
		return errs.Enrich(err, []string{
			"channels:history", "groups:history", "im:history", "mpim:history",
		})
	}

	if FlagIncludePermalinks {
		if err := permalink.EnrichMessages(ctx, cl, args[0], resp.Messages); err != nil {
			return err
		}
	}

	out := map[string]any{
		"channel":     args[0],
		"messages":    resp.Messages,
		"has_more":    resp.HasMore,
		"next_cursor": resp.ResponseMetaData.NextCursor,
		"pin_count":   resp.PinCount,
	}
	return output.Emit(out, fmtt, FlagOut)
}
