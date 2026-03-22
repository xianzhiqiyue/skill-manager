package handlers

import (
	"testing"

	"github.com/skill-home/server/internal/models"
)

func TestParsePaginationCapsPerPage(t *testing.T) {
	t.Parallel()

	page, perPage := parsePagination("3", "999")
	if page != 3 {
		t.Fatalf("unexpected page: %d", page)
	}
	if perPage != maxPerPage {
		t.Fatalf("expected perPage %d, got %d", maxPerPage, perPage)
	}
}

func TestValidatePathSegmentRejectsSlash(t *testing.T) {
	t.Parallel()

	if err := validatePathSegment("bad/name", "name"); err == nil {
		t.Fatal("expected error for slash in path segment")
	}
}

func TestPopulateSkillComputedFieldsSetsRating(t *testing.T) {
	t.Parallel()

	skill := models.Skill{
		RatingSum:   8,
		RatingCount: 2,
	}
	populateSkillComputedFields(&skill)
	if skill.Rating != 4 {
		t.Fatalf("unexpected computed rating: %v", skill.Rating)
	}
}
