package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Format string

const (
	FormatJSON   Format = "json"
	FormatPretty Format = "pretty"
)

func Print(w io.Writer, v any, format Format) error {
	enc := json.NewEncoder(w)
	switch format {
	case FormatJSON:
		return enc.Encode(v)
	case FormatPretty:
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	default:
		return fmt.Errorf("unknown output format: %s", format)
	}
}

// Emit writes v to stdout when outPath is empty, otherwise writes v to the
// given file and prints a small JSON summary ({"out","format","size_bytes"})
// to stdout. Parent directories are created as needed so callers can pass
// nested tmp paths like .claude/tmp/foo/bar.json without pre-mkdir.
func Emit(v any, format Format, outPath string) error {
	if outPath == "" {
		return Print(os.Stdout, v, format)
	}
	if dir := filepath.Dir(outPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create out dir: %w", err)
		}
	}
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create out file: %w", err)
	}
	defer f.Close()
	if err := Print(f, v, format); err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat out file: %w", err)
	}
	summary := map[string]any{
		"out":        outPath,
		"format":     string(format),
		"size_bytes": info.Size(),
	}
	return Print(os.Stdout, summary, FormatJSON)
}

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON, nil
	case "pretty":
		return FormatPretty, nil
	default:
		return "", fmt.Errorf("unknown output format: %s", s)
	}
}
