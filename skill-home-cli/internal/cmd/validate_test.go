package cmd

import "testing"

func TestValidateSkillRequiresCategoryAndOfficialTags(t *testing.T) {
	t.Parallel()

	content := []byte(`---
name: github
version: 1.0.0
description: GitHub skill
tags:
  - workflow
---

body
`)

	result := validateSkill(content, false)

	if len(result.Errors) == 0 {
		t.Fatal("expected validation errors")
	}
	if result.Errors[0].Field != "category" {
		t.Fatalf("expected first error to be category, got %#v", result.Errors)
	}
}

func TestValidateSkillRejectsUnknownCategoryAndIllegalTags(t *testing.T) {
	t.Parallel()

	content := []byte(`---
name: github
version: 1.0.0
description: GitHub skill
category: general
tags:
  - workflow
  - imported
  - deployment
  - ci-cd
  - review
---

body
`)

	result := validateSkill(content, false)

	if len(result.Errors) < 2 {
		t.Fatalf("expected multiple validation errors, got %#v", result.Errors)
	}
}

func TestValidateSkillAcceptsNewFixedCategory(t *testing.T) {
	t.Parallel()

	content := []byte(`---
name: crm-contract-audit
version: 1.0.0
description: Audit CRM contracts
category: 业务与管理
tags:
  - analysis
  - workflow
---

body
`)

	result := validateSkill(content, false)
	if len(result.Errors) != 0 {
		t.Fatalf("expected fixed Chinese category to pass, got %#v", result.Errors)
	}
}
