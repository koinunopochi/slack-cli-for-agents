package cmd

import (
	"context"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"

	"github.com/koinunopochi/slack-cli-for-agents/internal/client"
	"github.com/koinunopochi/slack-cli-for-agents/internal/config"
	"github.com/koinunopochi/slack-cli-for-agents/internal/errs"
	"github.com/koinunopochi/slack-cli-for-agents/internal/output"
	"github.com/koinunopochi/slack-cli-for-agents/internal/permalink"
)

var (
	readThreadLimit           int
	readThreadCursor          string
	readThreadOldest          string
	readThreadLatest          string
	readThreadInclusive       bool
	readThreadIncludeMetadata bool
	readThreadExcludeParent   bool
)

var readThreadCmd = &cobra.Command{
	Use:   "read-thread <channel-id> <thread-ts>",
	Short: "Fetch all replies in a Slack thread",
	Long: `Fetch the parent message and all replies for a given Slack thread.
Wraps the conversations.replies Web API. Supports pagination via --cursor /
--limit and time-range filtering via --oldest / --latest. Use --exclude-parent
to drop the parent message from the output when you only want the replies.`,
	Args: cobra.ExactArgs(2),
	RunE: runReadThread,
}

func init() {
	readThreadCmd.Flags().IntVar(&readThreadLimit, "limit", 200, "replies per request (recommended max 200)")
	readThreadCmd.Flags().StringVar(&readThreadCursor, "cursor", "", "pagination cursor")
	readThreadCmd.Flags().StringVar(&readThreadOldest, "oldest", "", "Slack ts; only newer than this")
	readThreadCmd.Flags().StringVar(&readThreadLatest, "latest", "", "Slack ts; only older than this")
	readThreadCmd.Flags().BoolVar(&readThreadInclusive, "inclusive", false, "include messages exactly at --oldest / --latest")
	readThreadCmd.Flags().BoolVar(&readThreadIncludeMetadata, "include-metadata", false, "include all message metadata")
	readThreadCmd.Flags().BoolVar(&readThreadExcludeParent, "exclude-parent", false, "exclude the parent message from output")
	attachDocumentation(readThreadCmd, commandDocs("read-thread"))
	RootCmd.AddCommand(readThreadCmd)
}

func runReadThread(c *cobra.Command, args []string) error {
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

	params := &slack.GetConversationRepliesParameters{
		ChannelID:          args[0],
		Timestamp:          args[1],
		Cursor:             readThreadCursor,
		Inclusive:          readThreadInclusive,
		Latest:             readThreadLatest,
		Limit:              readThreadLimit,
		Oldest:             readThreadOldest,
		IncludeAllMetadata: readThreadIncludeMetadata,
	}
	messages, hasMore, nextCursor, err := cl.GetConversationRepliesContext(ctx, params)
	if err != nil {
		return errs.Enrich(err, []string{
			"channels:history", "groups:history", "im:history", "mpim:history",
		})
	}

	if readThreadExcludeParent && len(messages) > 0 && messages[0].Timestamp == args[1] {
		messages = messages[1:]
	}

	if FlagIncludePermalinks {
		if err := permalink.EnrichMessages(ctx, cl, args[0], messages); err != nil {
			return err
		}
	}

	out := map[string]any{
		"channel":     args[0],
		"thread_ts":   args[1],
		"messages":    messages,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	}
	return output.Emit(out, fmtt, FlagOut)
}
