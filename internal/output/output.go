package output

import (
	"encoding/json"
	"fmt"
	"io"
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
