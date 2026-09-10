package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestHelpAndVersion(t *testing.T) {
	var out bytes.Buffer
	if code := Run([]string{"dc", "--help"}, os.Stdin, &out, &out); code != 0 {
		t.Fatalf("help %d", code)
	}
	if !strings.Contains(out.String(), "update") {
		t.Fatalf("help %s", out.String())
	}
	out.Reset()
	if code := Run([]string{"DiscordCli", "--version"}, os.Stdin, &out, &out); code != 0 {
		t.Fatalf("version %d", code)
	}
	if !strings.Contains(out.String(), "DiscordCli") || !strings.Contains(out.String(), Version) {
		t.Fatalf("version %s", out.String())
	}
}

func TestInviteWithoutID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var errBuf bytes.Buffer
	if code := Run([]string{"dc", "invite"}, os.Stdin, os.Stdout, &errBuf); code != 1 {
		t.Fatalf("code %d", code)
	}
	if !strings.Contains(errBuf.String(), "No bot id") {
		t.Fatalf("%s", errBuf.String())
	}
}

func TestLogout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	var out bytes.Buffer
	if code := Run([]string{"dc", "logout"}, os.Stdin, &out, &out); code != 0 {
		t.Fatalf("code %d %s", code, out.String())
	}
	if !strings.Contains(strings.ToLower(out.String()), "removed") {
		t.Fatalf("%s", out.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var errBuf bytes.Buffer
	if code := Run([]string{"dc", "nope"}, os.Stdin, os.Stdout, &errBuf); code != 2 {
		t.Fatalf("code %d", code)
	}
}
