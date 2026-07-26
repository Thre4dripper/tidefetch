package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPasswordFromEnvironmentReadsSecretFile(t *testing.T) {
	t.Setenv("TIDEFETCH_PASSWORD", "")
	path := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(path, []byte("from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TIDEFETCH_PASSWORD_FILE", path)

	password, err := passwordFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if password != "from-file" {
		t.Fatalf("password = %q, want from-file", password)
	}
}

func TestPasswordFromEnvironmentPrefersDirectValue(t *testing.T) {
	t.Setenv("TIDEFETCH_PASSWORD", "from-env")
	t.Setenv("TIDEFETCH_PASSWORD_FILE", "/missing")

	password, err := passwordFromEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	if password != "from-env" {
		t.Fatalf("password = %q, want from-env", password)
	}
}
