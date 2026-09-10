package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func isolate(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("PLAINCORD_TOKEN", "")
}

func TestLooksLikeToken(t *testing.T) {
	if !LooksLikeToken("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx") {
		t.Fatal("expected valid")
	}
	if LooksLikeToken("short") {
		t.Fatal("short token")
	}
	if LooksLikeToken("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx has space") {
		t.Fatal("space")
	}
}

func TestSaveLoadDeleteToken(t *testing.T) {
	isolate(t)
	token := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	if err := SaveToken(token); err != nil {
		t.Fatal(err)
	}
	path, err := TokenPath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	got, err := LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	if got != token {
		t.Fatalf("got %q", got)
	}
	if err := DeleteToken(); err != nil {
		t.Fatal(err)
	}
	got, err = LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestEnvTokenWins(t *testing.T) {
	isolate(t)
	if err := SaveToken("BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PLAINCORD_TOKEN", "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC")
	got, err := LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	if got != "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC" {
		t.Fatalf("got %q", got)
	}
}

func TestRejectShortToken(t *testing.T) {
	isolate(t)
	if err := SaveToken("nope"); err != ErrInvalidToken {
		t.Fatalf("got %v", err)
	}
}

func TestApplicationID(t *testing.T) {
	isolate(t)
	id, err := LoadApplicationID()
	if err != nil {
		t.Fatal(err)
	}
	if id != "" {
		t.Fatalf("got %q", id)
	}
	if err := SaveApplicationID("42"); err != nil {
		t.Fatal(err)
	}
	id, err = LoadApplicationID()
	if err != nil {
		t.Fatal(err)
	}
	if id != "42" {
		t.Fatalf("got %q", id)
	}
	path, _ := ConfigPath()
	if filepath.Base(path) != "config.json" {
		t.Fatalf("path %s", path)
	}
}
