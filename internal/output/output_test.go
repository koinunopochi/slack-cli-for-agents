package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Print(&buf, map[string]any{"k": "v"}, FormatJSON); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "{\"k\":\"v\"}\n" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestEmitToStdoutWhenNoOutPath(t *testing.T) {
	// We can't easily capture os.Stdout from inside the package without
	// extra wiring; instead just confirm Emit("") routes through Print by
	// having it not touch the filesystem.
	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	r, w, _ := os.Pipe()
	saved := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = saved })

	go func() {
		_ = Emit(map[string]any{"hello": "world"}, FormatJSON, "")
		_ = w.Close()
	}()
	out, _ := io.ReadAll(r)

	if string(out) != "{\"hello\":\"world\"}\n" {
		t.Errorf("unexpected stdout: %q", out)
	}
	entries, _ := os.ReadDir(tmp)
	if len(entries) != 0 {
		t.Errorf("Emit with empty outPath must not create files; got %d entries", len(entries))
	}
}

func TestEmitWritesFileAndPrintsSummary(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "nested", "payload.json")

	r, w, _ := os.Pipe()
	saved := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = saved })

	payload := map[string]any{"items": []int{1, 2, 3}}
	done := make(chan error, 1)
	go func() {
		done <- Emit(payload, FormatJSON, outPath)
		_ = w.Close()
	}()
	stdoutBytes, _ := io.ReadAll(r)
	if err := <-done; err != nil {
		t.Fatalf("Emit: %v", err)
	}

	// File must exist with the actual payload.
	body, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("file is not valid JSON: %v", err)
	}
	if items, ok := got["items"].([]any); !ok || len(items) != 3 {
		t.Errorf("payload mismatch: %v", got)
	}

	// Stdout must carry a small summary, not the full payload.
	var summary map[string]any
	if err := json.Unmarshal(stdoutBytes, &summary); err != nil {
		t.Fatalf("stdout summary not JSON: %v (raw=%q)", err, stdoutBytes)
	}
	if summary["out"] != outPath {
		t.Errorf("summary.out = %v, want %v", summary["out"], outPath)
	}
	if summary["format"] != "json" {
		t.Errorf("summary.format = %v, want json", summary["format"])
	}
	if _, ok := summary["size_bytes"]; !ok {
		t.Errorf("summary missing size_bytes: %v", summary)
	}
}
