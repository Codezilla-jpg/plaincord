package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUninstallRemovesBinsAndConfig(t *testing.T) {
	bin := t.TempDir()
	cfg := filepath.Join(t.TempDir(), "plaincord")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "dis"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "DiscordCli"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "dc"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "keepme"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg, "token"), []byte("tok"), 0o600); err != nil {
		t.Fatal(err)
	}
	removed, err := Uninstall(bin, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 4 {
		t.Fatalf("removed %v", removed)
	}
	for _, name := range []string{"dis", "DiscordCli", "dc"} {
		if _, err := os.Stat(filepath.Join(bin, name)); !os.IsNotExist(err) {
			t.Fatalf("%s still there: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(bin, "keepme")); err != nil {
		t.Fatal("keepme")
	}
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Fatalf("config still there: %v", err)
	}
}

func TestUninstallSkipsMissing(t *testing.T) {
	bin := t.TempDir()
	cfg := filepath.Join(t.TempDir(), "plaincord")
	removed, err := Uninstall(bin, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 0 {
		t.Fatalf("removed %v", removed)
	}
}

func TestUninstallIgnoresOtherConfigDir(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "notplaincord")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	removed, err := Uninstall(t.TempDir(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cfg); err != nil {
		t.Fatal("should keep foreign config")
	}
	if len(removed) != 0 {
		t.Fatalf("removed %v", removed)
	}
}
