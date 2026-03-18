package mailer

import (
	"strings"
	"testing"
)

func TestEncodeHeaderValueLeavesASCIIUntouched(t *testing.T) {
	t.Parallel()

	got := encodeHeaderValue("Daily Briefing")
	if got != "Daily Briefing" {
		t.Fatalf("expected ASCII header to remain unchanged, got %q", got)
	}
}

func TestEncodeHeaderValueEncodesUnicodeAndStripsNewlines(t *testing.T) {
	t.Parallel()

	got := encodeHeaderValue("NewsPocket 每日简报\r\nInjected")
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("expected encoded header to have no CR/LF, got %q", got)
	}
	if !strings.HasPrefix(got, "=?utf-8?") {
		t.Fatalf("expected RFC 2047 encoded header, got %q", got)
	}
}
