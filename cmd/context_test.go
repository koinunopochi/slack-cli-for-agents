package cmd

import (
	"context"
	"testing"

	"github.com/koinunopochi/slack-cli-for-agents/internal/contextcache"
	"github.com/spf13/cobra"
)

func TestEnsureContextForCommandPreflightBoundary(t *testing.T) {
	original := ensureContextCache
	t.Cleanup(func() { ensureContextCache = original })

	calls := 0
	ensureContextCache = func(context.Context, contextcache.SearchClient, contextcache.Options) (contextcache.Result, error) {
		calls++
		return contextcache.Result{}, nil
	}
	t.Setenv("SLACK_USER_TOKEN", "test-token")
	FlagTimeout = 0

	FlagTokenType = "user"
	if err := ensureContextForCommand(newTestCommand("read-channel"), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("user command preflight calls = %d, want 1", calls)
	}

	FlagTokenType = "bot"
	if err := ensureContextForCommand(newTestCommand("read-channel"), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("bot command preflight calls = %d, want unchanged", calls)
	}

	FlagTokenType = "user"
	for _, name := range []string{"context", "help", "completion"} {
		if err := ensureContextForCommand(newTestCommand(name), nil); err != nil {
			t.Fatalf("%s skip returned error: %v", name, err)
		}
	}
	if calls != 1 {
		t.Fatalf("skip command preflight calls = %d, want unchanged", calls)
	}
}

func newTestCommand(name string) *cobra.Command {
	return &cobra.Command{Use: name}
}
