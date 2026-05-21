package cmd

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
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
	resolveLimit           int
	resolveIncludeMetadata bool
)

var resolveCmd = &cobra.Command{
	Use:   "resolve <permalink>",
	Short: "Resolve a Slack permalink and fetch the enclosing thread",
	Long: `Resolve a Slack message permalink to its (channel, ts) tuple and fetch the
message plus any thread replies via conversations.replies.

Accepted formats:
  https://<workspace>.slack.com/archives/<CHANNEL>/p<TS_NO_DOT>
  https://<workspace>.slack.com/archives/<CHANNEL>/p<TS_NO_DOT>?thread_ts=<TS>&cid=...

When thread_ts is present the entire enclosing thread is returned; otherwise
the resolved ts is used as the thread anchor, which returns the message itself
plus its replies if it happens to be a parent.`,
	Args: cobra.ExactArgs(1),
	RunE: runResolve,
}

func init() {
	resolveCmd.Flags().IntVar(&resolveLimit, "limit", 200, "replies per request (recommended max 200)")
	resolveCmd.Flags().BoolVar(&resolveIncludeMetadata, "include-metadata", false, "include all message metadata")
	RootCmd.AddCommand(resolveCmd)
}

var permalinkRe = regexp.MustCompile(`/archives/([A-Z0-9]+)/p(\d+)`)

// parsePermalink extracts (channel, ts, thread_ts) from a Slack message
// permalink. Slack strips the dot from the ts in the URL path (e.g.
// 1716000000.123456 becomes p1716000000123456); we re-insert it so the value
// can be passed to conversations.replies and chat.getPermalink.
func parsePermalink(raw string) (channel, ts, threadTS string, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", "", fmt.Errorf("parse permalink: %w", err)
	}
	m := permalinkRe.FindStringSubmatch(u.Path)
	if m == nil {
		return "", "", "", fmt.Errorf("not a Slack message permalink (path=%q)", u.Path)
	}
	channel = m[1]
	tsRaw := m[2]
	if len(tsRaw) >= 7 {
		ts = tsRaw[:len(tsRaw)-6] + "." + tsRaw[len(tsRaw)-6:]
	} else {
		ts = tsRaw
	}
	threadTS = u.Query().Get("thread_ts")
	return channel, ts, threadTS, nil
}

func runResolve(c *cobra.Command, args []string) error {
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

	channel, ts, threadTS, err := parsePermalink(args[0])
	if err != nil {
		return err
	}
	anchor := threadTS
	if anchor == "" {
		anchor = ts
	}

	cl := client.New(token, client.Options{Timeout: FlagTimeout, Debug: FlagDebug})

	ctx, cancel := context.WithTimeout(c.Context(), FlagTimeout+5*time.Second)
	defer cancel()

	params := &slack.GetConversationRepliesParameters{
		ChannelID:          channel,
		Timestamp:          anchor,
		Limit:              resolveLimit,
		IncludeAllMetadata: resolveIncludeMetadata,
	}
	messages, hasMore, nextCursor, err := cl.GetConversationRepliesContext(ctx, params)
	if err != nil {
		return errs.Enrich(err, []string{
			"channels:history", "groups:history", "im:history", "mpim:history",
		})
	}

	if FlagIncludePermalinks {
		if err := permalink.EnrichMessages(ctx, cl, channel, messages); err != nil {
			return err
		}
	}

	out := map[string]any{
		"permalink":   args[0],
		"channel":     channel,
		"ts":          ts,
		"thread_ts":   threadTS,
		"messages":    messages,
		"has_more":    hasMore,
		"next_cursor": nextCursor,
	}
	return output.Emit(out, fmtt, FlagOut)
}
