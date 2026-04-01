package taxonomy

import "testing"

func TestLoadReturnsExpectedCategoriesAndTags(t *testing.T) {
	t.Parallel()

	definition, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(definition.Categories) != 8 {
		t.Fatalf("expected 8 categories, got %d", len(definition.Categories))
	}

	if !definition.HasCategory("ops") {
		t.Fatalf("expected ops category to exist")
	}

	if !definition.HasOfficialTag("ci-cd") {
		t.Fatalf("expected ci-cd official tag to exist")
	}

	if !definition.HasOfficialTag("deployment") {
		t.Fatalf("expected deployment official tag to exist")
	}
}

func TestNormalizeTagCanonicalizesAliases(t *testing.T) {
	t.Parallel()

	definition, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	tests := map[string]string{
		"pipeline":   "ci-cd",
		"CI":         "ci-cd",
		"deploy":     "deployment",
		" workflow ": "workflow",
	}

	for input, want := range tests {
		if got := definition.NormalizeTag(input); got != want {
			t.Fatalf("NormalizeTag(%q) = %q, want %q", input, got, want)
		}
	}
}

