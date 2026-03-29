package config

import "testing"

func TestLoadFromEnvSetsStoragePublicBaseURL(t *testing.T) {
	t.Setenv("SKILL_HOME_STORAGE_PUBLIC_BASE_URL", "https://cdn.example.com/skill-home-assets")

	cfg = &Config{}
	loadFromEnv()

	if got, want := cfg.Storage.PublicBaseURL, "https://cdn.example.com/skill-home-assets"; got != want {
		t.Fatalf("cfg.Storage.PublicBaseURL = %q, want %q", got, want)
	}
}
