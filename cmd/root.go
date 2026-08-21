package cmd

import (
	"time"

	"github.com/spf13/cobra"
)

const (
	documentationURL        = "https://github.com/koinunopochi/slack-cli-for-agents/blob/main/docs/README.md"
	commandDocumentationURL = "https://github.com/koinunopochi/slack-cli-for-agents/blob/main/docs/commands.md"
)

var (
	FlagTokenType         string
	FlagFormat            string
	FlagTimeout           time.Duration
	FlagDebug             bool
	FlagOut               string
	FlagIncludePermalinks bool
)

var RootCmd = &cobra.Command{
	Use:   "slack",
	Short: "Slack Web API CLI for AI agents",
	Long: `slack is a thin Slack Web API CLI tailored for AI agents.
It exposes read-only collection commands (read-channel, read-thread, resolve,
search-messages, search-files, search-channels, search-users, user-activity)
and emits JSON by default so downstream agents can consume the output without
extra parsing. Use --out <path> to keep large payloads out of the agent's
context window.`,
	SilenceUsage:  true,
	SilenceErrors: false,
}

func init() {
	attachDocumentation(RootCmd, documentationURL)
	RootCmd.PersistentFlags().StringVarP(&FlagTokenType, "token-type", "t", "user",
		"Slack token type: user | bot")
	RootCmd.PersistentFlags().StringVarP(&FlagFormat, "format", "f", "json",
		"Output format: json | pretty")
	RootCmd.PersistentFlags().DurationVar(&FlagTimeout, "timeout", 30*time.Second,
		"HTTP timeout for each Slack API call")
	RootCmd.PersistentFlags().BoolVar(&FlagDebug, "debug", false,
		"Enable slack-go debug logging to stderr")
	RootCmd.PersistentFlags().StringVar(&FlagOut, "out", "",
		"Write payload to this file instead of stdout; stdout receives a small JSON summary "+
			"({out, format, size_bytes}). Parent dirs are created automatically.")
	RootCmd.PersistentFlags().BoolVar(&FlagIncludePermalinks, "include-permalinks", false,
		"Populate the permalink field on each message via chat.getPermalink. "+
			"Costs one extra API call per message; search.messages already returns permalinks "+
			"so this is a no-op there.")
}

func attachDocumentation(command *cobra.Command, target string) {
	command.Long += "\n\nDetailed documentation:\n  " + target
}

func commandDocs(section string) string {
	return commandDocumentationURL + "#" + section
}

func Execute() error { return RootCmd.Execute() }
