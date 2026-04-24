package httpclient

import (
	"errors"
	"testing"
)

func TestIsBotBlock(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{"200 normal", 200, "normal page content", false},
		{"503 plain", 503, "Service Unavailable", false},
		{"503 cloudflare word", 503, "<html>cloudflare protection</html>", true},
		{"503 just a moment", 503, "Just a moment... enable javascript", true},
		{"503 cf-ray header in body", 503, "CF-Ray: 1234abcd", true},
		{"403 challenge page", 403, "Attention required! challenge pending", true},
		{"403 enable javascript", 403, "Please enable JavaScript and cookies", true},
		{"404 not found", 404, "404 Not Found", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isBotBlock(tc.status, []byte(tc.body))
			if got != tc.want {
				t.Errorf("isBotBlock(%d, %q) = %v, want %v", tc.status, tc.body, got, tc.want)
			}
		})
	}
}

func TestErrBotBlockSentinel(t *testing.T) {
	if !errors.Is(ErrBotBlock, ErrBotBlock) {
		t.Error("ErrBotBlock should satisfy errors.Is with itself")
	}
	if ErrBotBlock.Error() == "" {
		t.Error("ErrBotBlock should have a non-empty message")
	}
}
