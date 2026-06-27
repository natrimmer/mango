package main

import (
	"strings"
	"testing"
)

func TestVersionStringHidesUnknownFields(t *testing.T) {
	version, buildDate, commitSHA = "v1.0.3", "unknown", "unknown"
	out := versionString()
	if strings.Contains(out, "Build Date") || strings.Contains(out, "Commit") {
		t.Errorf("unknown date/commit should be omitted, got:\n%s", out)
	}

	version, buildDate, commitSHA = "v1.0.3", "2026-06-27T00:00:00Z", "abc1234"
	out = versionString()
	if !strings.Contains(out, "Build Date: 2026-06-27") || !strings.Contains(out, "Commit: abc1234") {
		t.Errorf("known date/commit should be shown, got:\n%s", out)
	}
}
