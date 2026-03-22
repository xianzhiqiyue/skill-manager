package models

import "testing"

func TestSkillGetRating(t *testing.T) {
	t.Parallel()

	skill := Skill{
		RatingSum:   9,
		RatingCount: 2,
	}

	if got := skill.GetRating(); got != 4.5 {
		t.Fatalf("unexpected rating: %v", got)
	}
}

func TestSkillGetRatingZeroCount(t *testing.T) {
	t.Parallel()

	skill := Skill{}
	if got := skill.GetRating(); got != 0 {
		t.Fatalf("expected zero rating, got %v", got)
	}
}
