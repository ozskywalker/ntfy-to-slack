package slack

import (
	"testing"
	"time"
)

// TestBackoffDelay_CapsAtMaxBackoff exercises backoffDelay directly, since
// Send's own retry loop (maxSendAttempts is small) never runs enough
// attempts to reach the cap itself.
func TestBackoffDelay_CapsAtMaxBackoff(t *testing.T) {
	tests := []struct {
		name    string
		attempt int
		want    time.Duration
	}{
		{"first retry uses base delay", 1, baseBackoff},
		{"second retry doubles", 2, baseBackoff * 2},
		{"large attempt caps at maxBackoff", 10, maxBackoff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := backoffDelay(tt.attempt)
			if got != tt.want {
				t.Errorf("backoffDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name string
		v    string
		want time.Duration
	}{
		{"empty header falls back to 0", "", 0},
		{"non-numeric header falls back to 0", "not-a-number", 0},
		{"negative header falls back to 0", "-5", 0},
		{"zero header falls back to 0", "0", 0},
		{"valid header is honored", "30", 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRetryAfter(tt.v)
			if got != tt.want {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.v, got, tt.want)
			}
		})
	}
}
