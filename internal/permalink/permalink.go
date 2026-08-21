// Package permalink fills in the Permalink field on Slack messages by calling
// chat.getPermalink for each message that does not already have one.
//
// search.messages already returns permalinks, but conversations.history and
// conversations.replies do not, so callers that want stable cross-references
// in their output enable this enrichment via --include-permalinks.
package permalink

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"
)

// EnrichMessages populates msgs[i].Permalink for each entry whose Permalink
// is currently empty, by calling chat.getPermalink against the given channel.
// Messages that already carry a Permalink (e.g. from search.messages) or that
// have no Timestamp are skipped.
//
// On the first failed call the partially enriched slice is returned along
// with the error; callers can still emit what was collected so far.
func EnrichMessages(ctx context.Context, cl *slack.Client, channel string, msgs []slack.Message) error {
	for i := range msgs {
		if msgs[i].Permalink != "" || msgs[i].Timestamp == "" {
			continue
		}
		link, err := cl.GetPermalinkContext(ctx, &slack.PermalinkParameters{
			Channel: channel,
			Ts:      msgs[i].Timestamp,
		})
		if err != nil {
			return fmt.Errorf("get permalink for %s/%s: %w", channel, msgs[i].Timestamp, err)
		}
		msgs[i].Permalink = link
	}
	return nil
}
