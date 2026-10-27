package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDotEnvUsesLocalFileWithoutOverridingHost(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	const localKey = "MATCHMATES_TEST_LOCAL_SETTING"
	if _, exists := os.LookupEnv(localKey); exists {
		t.Fatalf("unexpected preexisting %s", localKey)
	}
	t.Cleanup(func() { _ = os.Unsetenv(localKey) })
	t.Setenv("TMDB_API_KEY", "host-key")
	if err := os.WriteFile(path, []byte("# local settings\nTMDB_API_KEY=file-key\n"+localKey+"=\"local-value\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadDotEnv(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("TMDB_API_KEY"); got != "host-key" {
		t.Fatal("host environment was overwritten")
	}
	if got := os.Getenv(localKey); got != "local-value" {
		t.Fatalf("local setting = %q", got)
	}
	if err := loadDotEnv(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("missing local file: %v", err)
	}
	if err := os.WriteFile(path, []byte("bad line"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := loadDotEnv(path); err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("invalid file error = %v", err)
	}
}
