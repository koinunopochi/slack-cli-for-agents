package client

import (
	"net/http"
	"time"

	"github.com/slack-go/slack"
)

type Options struct {
	Timeout    time.Duration
	Debug      bool
	MaxRetries int
}

func New(token string, opts Options) *slack.Client {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	cfg := slack.DefaultRetryConfig()
	if opts.MaxRetries > 0 {
		cfg.MaxRetries = opts.MaxRetries
	}
	cfg.Handlers = slack.AllBuiltinRetryHandlers(cfg)

	slackOpts := []slack.Option{
		slack.OptionHTTPClient(&http.Client{Timeout: timeout}),
		slack.OptionRetryConfig(cfg),
	}
	if opts.Debug {
		slackOpts = append(slackOpts, slack.OptionDebug(true))
	}

	return slack.New(token, slackOpts...)
}
