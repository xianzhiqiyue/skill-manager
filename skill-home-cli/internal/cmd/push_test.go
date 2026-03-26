package cmd

import "testing"

func TestPushHelpDoesNotPanicFromFlagConflicts(t *testing.T) {
	t.Parallel()

	root := NewRootCmd("test", "test", "test")
	root.SetArgs([]string{"push", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
}
