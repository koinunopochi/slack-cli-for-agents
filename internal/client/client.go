package client

import (
	"net/http"
	"sync"
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

	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: &scopeCapturingTransport{inner: http.DefaultTransport},
	}

	slackOpts := []slack.Option{
		slack.OptionHTTPClient(httpClient),
		slack.OptionRetryConfig(cfg),
	}
	if opts.Debug {
		slackOpts = append(slackOpts, slack.OptionDebug(true))
	}

	return slack.New(token, slackOpts...)
}

var (
	lastScopesMu sync.Mutex
	lastScopes   string
)

// LastScopes returns the most recent value of the `x-oauth-scopes` response
// header observed on a Slack API call made through this client. Slack returns
// this header on every authenticated response; we capture it so that error
// diagnostics can show the caller what scopes their token actually has when a
// missing_scope failure occurs.
func LastScopes() string {
	lastScopesMu.Lock()
	defer lastScopesMu.Unlock()
	return lastScopes
}

type scopeCapturingTransport struct {
	inner http.RoundTripper
}

func (t *scopeCapturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.inner.RoundTrip(req)
	if resp != nil {
		if s := resp.Header.Get("x-oauth-scopes"); s != "" {
			lastScopesMu.Lock()
			lastScopes = s
			lastScopesMu.Unlock()
		}
	}
	return resp, err
}
