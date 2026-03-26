package cmd

import "testing"

func TestRootCommandIncludesSelfUpdate(t *testing.T) {
	t.Parallel()

	rootCmd := NewRootCmd("v1.2.3", "abc123", "2026-03-26T00:00:00Z")
	found, _, err := rootCmd.Find([]string{"self-update"})
	if err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	if found == nil {
		t.Fatal("self-update command not found")
	}
	if got := found.Name(); got != "self-update" {
		t.Fatalf("command name = %q, want %q", got, "self-update")
	}
}
