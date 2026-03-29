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

func TestLoadFromEnvFallsBackToSoulstoreSkillOSSValues(t *testing.T) {
	t.Setenv("SOULSTORE_SKILL_OSS_INTERNAL_ENDPOINT", "https://oss-cn-hangzhou-internal.aliyuncs.com")
	t.Setenv("SOULSTORE_SKILL_OSS_PUBLIC_CNAME_ENDPOINT", "skill-home.ciqtek.com")
	t.Setenv("SOULSTORE_SKILL_OSS_ACCESS_KEY_ID", "test-ak")
	t.Setenv("SOULSTORE_SKILL_OSS_ACCESS_KEY_SECRET", "test-sk")
	t.Setenv("SOULSTORE_SKILL_OSS_BUCKET_RELEASE", "skill-home")
	t.Setenv("SOULSTORE_SKILL_OSS_REGION", "cn-hangzhou")
	t.Setenv("SOULSTORE_SKILL_OSS_USE_INTERNAL_ENDPOINT", "true")

	cfg = &Config{}
	loadFromEnv()

	if got, want := cfg.Storage.Type, "s3"; got != want {
		t.Fatalf("cfg.Storage.Type = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.Endpoint, "oss-cn-hangzhou-internal.aliyuncs.com"; got != want {
		t.Fatalf("cfg.Storage.Endpoint = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.AccessKey, "test-ak"; got != want {
		t.Fatalf("cfg.Storage.AccessKey = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.SecretKey, "test-sk"; got != want {
		t.Fatalf("cfg.Storage.SecretKey = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.Bucket, "skill-home"; got != want {
		t.Fatalf("cfg.Storage.Bucket = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.Region, "cn-hangzhou"; got != want {
		t.Fatalf("cfg.Storage.Region = %q, want %q", got, want)
	}
	if !cfg.Storage.UseSSL {
		t.Fatal("cfg.Storage.UseSSL = false, want true")
	}
	if got, want := cfg.Storage.PublicBaseURL, "https://skill-home.ciqtek.com"; got != want {
		t.Fatalf("cfg.Storage.PublicBaseURL = %q, want %q", got, want)
	}
}

func TestLoadFromEnvPrefersSkillHomeStorageValuesOverSoulstoreFallback(t *testing.T) {
	t.Setenv("SKILL_HOME_STORAGE_TYPE", "minio")
	t.Setenv("SKILL_HOME_STORAGE_ENDPOINT", "localhost:19000")
	t.Setenv("SKILL_HOME_STORAGE_ACCESS_KEY", "minioadmin")
	t.Setenv("SKILL_HOME_STORAGE_SECRET_KEY", "miniosecret")
	t.Setenv("SKILL_HOME_STORAGE_BUCKET", "local-bucket")
	t.Setenv("SKILL_HOME_STORAGE_REGION", "local-region")
	t.Setenv("SKILL_HOME_STORAGE_USE_SSL", "false")
	t.Setenv("SKILL_HOME_STORAGE_PUBLIC_BASE_URL", "https://explicit.example.com/root")

	t.Setenv("SOULSTORE_SKILL_OSS_INTERNAL_ENDPOINT", "oss-cn-hangzhou-internal.aliyuncs.com")
	t.Setenv("SOULSTORE_SKILL_OSS_PUBLIC_CNAME_ENDPOINT", "skill-home.ciqtek.com")
	t.Setenv("SOULSTORE_SKILL_OSS_ACCESS_KEY_ID", "test-ak")
	t.Setenv("SOULSTORE_SKILL_OSS_ACCESS_KEY_SECRET", "test-sk")
	t.Setenv("SOULSTORE_SKILL_OSS_BUCKET_RELEASE", "skill-home")
	t.Setenv("SOULSTORE_SKILL_OSS_REGION", "cn-hangzhou")
	t.Setenv("SOULSTORE_SKILL_OSS_USE_INTERNAL_ENDPOINT", "true")

	cfg = &Config{}
	loadFromEnv()

	if got, want := cfg.Storage.Type, "minio"; got != want {
		t.Fatalf("cfg.Storage.Type = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.Endpoint, "localhost:19000"; got != want {
		t.Fatalf("cfg.Storage.Endpoint = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.AccessKey, "minioadmin"; got != want {
		t.Fatalf("cfg.Storage.AccessKey = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.SecretKey, "miniosecret"; got != want {
		t.Fatalf("cfg.Storage.SecretKey = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.Bucket, "local-bucket"; got != want {
		t.Fatalf("cfg.Storage.Bucket = %q, want %q", got, want)
	}
	if got, want := cfg.Storage.Region, "local-region"; got != want {
		t.Fatalf("cfg.Storage.Region = %q, want %q", got, want)
	}
	if cfg.Storage.UseSSL {
		t.Fatal("cfg.Storage.UseSSL = true, want false")
	}
	if got, want := cfg.Storage.PublicBaseURL, "https://explicit.example.com/root"; got != want {
		t.Fatalf("cfg.Storage.PublicBaseURL = %q, want %q", got, want)
	}
}
