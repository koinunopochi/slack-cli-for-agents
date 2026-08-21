// Package errs decorates Slack API errors with extra diagnostics so users do
// not have to dig through Slack's docs to understand why a call failed.
//
// At the moment the only case we enrich is missing_scope, because that is
// the failure that hurts the most for new tokens: Slack's reply is literally
// just "missing_scope" with no hint of which scope is missing or which scopes
// the token actually has. We pull the provided scopes from the captured
// x-oauth-scopes header (internal/client) and combine them with a CLI-side
// hint of what the failing command needs.
package errs

import (
	"errors"
	"fmt"
	"strings"

	"github.com/koinunopochi/slack-cli-for-agents/internal/client"
)

// Enrich returns err unchanged unless it represents a missing_scope failure,
// in which case it wraps the message with the CLI's best guess at the needed
// scopes (cmdNeeded), the scopes the token actually has (read from the most
// recent Slack response), and a one-line hint pointing the user at the Slack
// App OAuth settings.
//
// cmdNeeded should list scopes the command is *likely* to require given its
// arguments; it is a static hint, not a precise diagnosis, since Slack does
// not return the missing scope on every error path.
func Enrich(err error, cmdNeeded []string) error {
	if err == nil {
		return nil
	}
	if !strings.Contains(err.Error(), "missing_scope") {
		return err
	}

	var b strings.Builder
	b.WriteString(err.Error())
	if len(cmdNeeded) > 0 {
		fmt.Fprintf(&b, "\n  needed:   %s", strings.Join(cmdNeeded, ", "))
	}
	if provided := client.LastScopes(); provided != "" {
		fmt.Fprintf(&b, "\n  provided: %s", provided)
	}
	b.WriteString("\n  hint:     Add the missing scopes to your Slack App OAuth scopes (api.slack.com/apps) and reinstall the app to your workspace.")
	return errors.New(b.String())
}
