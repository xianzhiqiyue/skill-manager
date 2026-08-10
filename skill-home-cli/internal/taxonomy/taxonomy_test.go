package taxonomy

import "testing"

func TestLoadReturnsExpectedCategoriesAndTags(t *testing.T) {
	t.Parallel()

	definition, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(definition.Categories) != 12 {
		t.Fatalf("expected 12 categories, got %d", len(definition.Categories))
	}

	for _, category := range []string{"开发与编程", "数据与分析", "设计与内容", "业务与管理", "Agent 与 Skill 工具", "运维与安全"} {
		if !definition.HasCategory(category) {
			t.Fatalf("expected %s category to exist", category)
		}
	}

	if !definition.HasOfficialTag("ci-cd") {
		t.Fatalf("expected ci-cd official tag to exist")
	}

	if !definition.HasOfficialTag("deployment") {
		t.Fatalf("expected deployment official tag to exist")
	}
}

func TestNormalizeCategoryCanonicalizesLegacyEnglishValues(t *testing.T) {
	t.Parallel()

	definition, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	tests := map[string]string{
		"business":         "业务与管理",
		" OPS ":            "运维与安全",
		"开发与编程":            "开发与编程",
		"Agent 与 Skill 工具": "Agent 与 Skill 工具",
	}
	for input, want := range tests {
		if got := definition.NormalizeCategory(input); got != want {
			t.Fatalf("NormalizeCategory(%q) = %q, want %q", input, got, want)
		}
		if !definition.HasCategory(input) {
			t.Fatalf("expected %q to resolve to a fixed category", input)
		}
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
