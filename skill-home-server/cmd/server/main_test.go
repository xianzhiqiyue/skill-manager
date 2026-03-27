package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHandleVersionCommandPrintsBuildInfo(t *testing.T) {
	originalVersion, originalCommit, originalBuildDate := version, commit, buildDate
	version, commit, buildDate = "v9.9.9", "abc1234", "2026-03-27T03:00:00Z"
	t.Cleanup(func() {
		version, commit, buildDate = originalVersion, originalCommit, originalBuildDate
	})

	for _, args := range [][]string{{"--version"}, {"version"}} {
		var buffer bytes.Buffer

		handled, err := handleVersionCommand(args, &buffer)
		if err != nil {
			t.Fatalf("handleVersionCommand(%v) returned error: %v", args, err)
		}
		if !handled {
			t.Fatalf("handleVersionCommand(%v) = false, want true", args)
		}

		output := buffer.String()
		for _, want := range []string{"skill-home-server", "Version:   v9.9.9", "Commit:    abc1234", "BuildDate: 2026-03-27T03:00:00Z"} {
			if !strings.Contains(output, want) {
				t.Fatalf("output %q does not contain %q", output, want)
			}
		}
	}
}

func TestHandleVersionCommandIgnoresNonVersionArgs(t *testing.T) {
	var buffer bytes.Buffer

	handled, err := handleVersionCommand([]string{"serve"}, &buffer)
	if err != nil {
		t.Fatalf("handleVersionCommand returned error: %v", err)
	}
	if handled {
		t.Fatal("handleVersionCommand returned true for non-version args")
	}
	if buffer.Len() != 0 {
		t.Fatalf("buffer = %q, want empty", buffer.String())
	}
}
