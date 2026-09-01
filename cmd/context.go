package cmd

import (
	"fmt"
	"os"

	"github.com/slack-go/slack"
	"github.com/spf13/cobra"

	"github.com/koinunopochi/slack-cli-for-agents/internal/client"
	"github.com/koinunopochi/slack-cli-for-agents/internal/config"
	contextcache "github.com/koinunopochi/slack-cli-for-agents/internal/contextcache"
	"github.com/koinunopochi/slack-cli-for-agents/internal/errs"
	"github.com/koinunopochi/slack-cli-for-agents/internal/output"
)

var contextRefresh bool

var ensureContextCache = contextcache.Ensure

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Build or read the current user's Slack context cache",
	Long: `Build or read the current user's derived Slack context.

The first run searches the last 30 days of the authenticated user's messages
and stores only workspace/user IDs and per-channel activity aggregates in the
user cache. A cache older than 24 hours is rebuilt from the rolling 30-day
window. Message bodies and tokens are never stored.

This command requires a User Token because Slack's search.messages API is used
to discover the authenticated user's active channels.`,
	Args: cobra.NoArgs,
	RunE: runContext,
}

func init() {
	contextCmd.Flags().BoolVar(&contextRefresh, "refresh", false, "force a rolling 30-day rebuild now")
	attachDocumentation(contextCmd, commandDocs("context"))
	RootCmd.AddCommand(contextCmd)
}

func runContext(c *cobra.Command, _ []string) error {
	tt, err := config.ParseTokenType(FlagTokenType)
	if err != nil {
		return err
	}
	if tt != config.TokenTypeUser {
		return fmt.Errorf("context requires --token-type user (it discovers the authenticated user's activity)")
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
	result, err := ensureContextCache(c.Context(), cl, contextcache.Options{Force: contextRefresh})
	if err != nil {
		return errs.Enrich(err, []string{"search:read"})
	}
	return output.Emit(map[string]any{
		"status":        result.Status,
		"refreshed":     result.Refreshed,
		"refresh_error": result.RefreshError,
		"context":       result.Snapshot,
	}, fmtt, FlagOut)
}

func ensureContextForCommand(c *cobra.Command, _ []string) error {
	if c.Name() == "context" || c.Name() == "help" || c.Name() == "completion" {
		return nil
	}
	tt, err := config.ParseTokenType(FlagTokenType)
	if err != nil {
		return err
	}
	// A bot token can still read supported resources, but it cannot represent
	// the human user's activity that this cache describes.
	if tt != config.TokenTypeUser {
		return nil
	}
	token, err := config.LoadToken(tt)
	if err != nil {
		return err
	}
	cl := client.New(token, client.Options{Timeout: FlagTimeout, Debug: FlagDebug})
	result, err := ensureContextCache(c.Context(), cl, contextcache.Options{})
	if err != nil {
		return errs.Enrich(err, []string{"search:read"})
	}
	if result.Status == contextcache.StatusStale {
		fmt.Fprintf(os.Stderr, "warning: Slack context cache is stale; refresh failed (%s)\n", result.RefreshError)
	}
	if result.Status == contextcache.StatusPartial {
		fmt.Fprintln(os.Stderr, "warning: Slack context cache is partial; the search page limit was reached")
	}
	return nil
}

var _ contextcache.SearchClient = (*slack.Client)(nil)
