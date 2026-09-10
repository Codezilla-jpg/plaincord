package update

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func digestHex(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

func TestParseTag(t *testing.T) {
	got := ParseTag("https://github.com/Codezilla-jpg/plaincord/releases/download/v0.3.0/dis_darwin_arm64")
	if got != "v0.3.0" {
		t.Fatalf("got %q", got)
	}
}

func TestParseSums(t *testing.T) {
	body := digestHex("new-binary") + "  dis_linux_amd64\n"
	d, err := ParseSums(body, "dis_linux_amd64")
	if err != nil {
		t.Fatal(err)
	}
	if d != "sha256:"+digestHex("new-binary") {
		t.Fatalf("%s", d)
	}
}

func TestAssetName(t *testing.T) {
	if AssetName("linux", "amd64") != "dis_linux_amd64" {
		t.Fatal(AssetName("linux", "amd64"))
	}
	if AssetName("windows", "amd64") != "dis_windows_amd64.exe" {
		t.Fatal(AssetName("windows", "amd64"))
	}
}

func TestNeedsUpdate(t *testing.T) {
	if !NeedsUpdate("0.2.0", "v0.3.0") {
		t.Fatal("expected update")
	}
	if NeedsUpdate("v0.2.0", "0.2.0") {
		t.Fatal("same version")
	}
}

func testServer(t *testing.T, tag, body string, digestName string) *httptest.Server {
	t.Helper()
	name := AssetName(runtime.GOOS, runtime.GOARCH)
	if digestName == "" {
		digestName = name
	}
	sums := digestHex(body) + "  " + digestName + "\n"
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		latest := "/Codezilla-jpg/plaincord/releases/latest/download/" + name
		sumsPath := "/Codezilla-jpg/plaincord/releases/latest/download/SHA256SUMS"
		final := "/Codezilla-jpg/plaincord/releases/download/" + tag + "/" + name
		switch r.URL.Path {
		case latest:
			http.Redirect(w, r, final, http.StatusFound)
		case sumsPath:
			_, _ = w.Write([]byte(sums))
		case final:
			_, _ = w.Write([]byte(body))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestApplyReplacesBinary(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dis")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "new-binary"
	srv := testServer(t, "v0.3.0", body, "")
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), Base: srv.URL, Repo: DefaultRepo}
	tag, err := c.Apply("0.2.0", dest)
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v0.3.0" {
		t.Fatalf("tag %s", tag)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("got %q", got)
	}
}

func TestApplyRejectsBadDigest(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dis")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	name := AssetName(runtime.GOOS, runtime.GOARCH)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Codezilla-jpg/plaincord/releases/latest/download/"+name {
			http.Redirect(w, r, "/Codezilla-jpg/plaincord/releases/download/v0.9.0/"+name, http.StatusFound)
			return
		}
		if r.URL.Path == "/Codezilla-jpg/plaincord/releases/latest/download/SHA256SUMS" {
			_, _ = w.Write([]byte(digestHex("expected") + "  " + name + "\n"))
			return
		}
		_, _ = w.Write([]byte("tampered"))
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), Base: srv.URL, Repo: DefaultRepo}
	if _, err := c.Apply("0.2.0", dest); !errors.Is(err, ErrBadDigest) {
		t.Fatalf("got %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "old" {
		t.Fatalf("replaced: %q", got)
	}
}

func TestApplyNoUpdate(t *testing.T) {
	srv := testServer(t, "v0.2.0", "same", "")
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), Base: srv.URL, Repo: DefaultRepo}
	tag, err := c.Apply("0.2.0", filepath.Join(t.TempDir(), "dis"))
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v0.2.0" {
		t.Fatalf("tag %s", tag)
	}
}
