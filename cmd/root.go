package cmd

import (
	"time"

	"github.com/spf13/cobra"
)

var (
	FlagTokenType string
	FlagFormat    string
	FlagTimeout   time.Duration
	FlagDebug     bool
)

var RootCmd = &cobra.Command{
	Use:   "play-slack",
	Short: "Slack Web API CLI for AI agents",
	Long: `play-slack is a thin Slack Web API CLI tailored for AI agents.
It exposes read-only collection commands (read-channel, read-thread,
search-messages, search-channels, search-users) and emits JSON by default
so downstream agents can consume the output without extra parsing.`,
	SilenceUsage:  true,
	SilenceErrors: false,
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&FlagTokenType, "token-type", "t", "user",
		"Slack token type: user | bot")
	RootCmd.PersistentFlags().StringVarP(&FlagFormat, "format", "f", "json",
		"Output format: json | pretty")
	RootCmd.PersistentFlags().DurationVar(&FlagTimeout, "timeout", 30*time.Second,
		"HTTP timeout for each Slack API call")
	RootCmd.PersistentFlags().BoolVar(&FlagDebug, "debug", false,
		"Enable slack-go debug logging to stderr")
}

func Execute() error { return RootCmd.Execute() }
