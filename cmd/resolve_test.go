package cmd

import "testing"

func TestParsePermalink(t *testing.T) {
	cases := []struct {
		name        string
		in          string
		wantChannel string
		wantTS      string
		wantThread  string
		wantErr     bool
	}{
		{
			name:        "single message",
			in:          "https://example.slack.com/archives/C08LBTBLSMD/p1716000000123456",
			wantChannel: "C08LBTBLSMD",
			wantTS:      "1716000000.123456",
		},
		{
			name:        "thread reply",
			in:          "https://example.slack.com/archives/C08LBTBLSMD/p1716000001000000?thread_ts=1716000000.123456&cid=C08LBTBLSMD",
			wantChannel: "C08LBTBLSMD",
			wantTS:      "1716000001.000000",
			wantThread:  "1716000000.123456",
		},
		{
			name:        "private channel (G-prefix legacy)",
			in:          "https://example.slack.com/archives/GABCDEF12/p1716000000123456",
			wantChannel: "GABCDEF12",
			wantTS:      "1716000000.123456",
		},
		{
			name:    "non-archive URL",
			in:      "https://example.slack.com/team/U12345",
			wantErr: true,
		},
		{
			name:    "garbage input",
			in:      "not a url",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch, ts, thr, err := parsePermalink(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got channel=%q ts=%q thread=%q", ch, ts, thr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ch != tc.wantChannel || ts != tc.wantTS || thr != tc.wantThread {
				t.Fatalf("mismatch:\n  got  channel=%q ts=%q thread=%q\n  want channel=%q ts=%q thread=%q",
					ch, ts, thr, tc.wantChannel, tc.wantTS, tc.wantThread)
			}
		})
	}
}
