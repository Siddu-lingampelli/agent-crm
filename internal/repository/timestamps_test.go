package repository

import (
	"testing"
	"time"
)

func TestParseTimeCompatibility(t *testing.T) {
	for _, input := range []string{
		"2026-11-10T13:30:40.2445691Z",
		"2026-11-10 13:30:40.2445691 +0000 UTC",
		"2026-11-10 19:00:40.2445691 +0530 IST",
		"2026-11-10 13:30:40.2445691+00:00",
		"2026-11-10 13:30:40.2445691",
	} {
		t.Run(input, func(t *testing.T) {
			want, _ := time.Parse(time.RFC3339Nano, "2026-11-10T13:30:40.2445691Z")
			if got := parseTime(input); !got.Equal(want) || got.Location() != time.UTC {
				t.Fatalf("parsed %q as %s, want %s", input, got, want)
			}
		})
	}
	if !parseTime("invalid").IsZero() {
		t.Fatal("invalid timestamps should remain unavailable")
	}
}
