package cmd

import "testing"

func TestLooksLikeUserID(t *testing.T) {
	cases := map[string]bool{
		"U08L3MPJB9T":  true,
		"W08L3MPJB9T":  true,
		"UABCDEFG1":    true,
		"U12345678":    true,
		"Umberto":      false,
		"u08l3mpjb9t":  false,
		"U1234567":     false,
		"":             false,
		"okayama":      false,
		"U-1234567890": false,
	}
	for in, want := range cases {
		if got := looksLikeUserID(in); got != want {
			t.Errorf("looksLikeUserID(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNormalizeIn(t *testing.T) {
	cases := map[string]string{
		"general":          "#general",
		"#general":         "#general",
		"  #general  ":     "#general",
		"<#C0123|general>": "<#C0123|general>",
	}
	for in, want := range cases {
		if got := normalizeIn(in); got != want {
			t.Errorf("normalizeIn(%q) = %q, want %q", in, got, want)
		}
	}
}
