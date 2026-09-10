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

func digestOf(body string) string {
	sum := sha256.Sum256([]byte(body))
	return "sha256:" + hex.EncodeToString(sum[:])
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
	if NeedsUpdate("0.2.0", "") {
		t.Fatal("empty latest")
	}
}

func TestApplyReplacesBinary(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dis")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "new-binary"
	wantName := AssetName(runtime.GOOS, runtime.GOARCH)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/Codezilla-jpg/plaincord/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v0.3.0","assets":[{"name":"` + wantName + `","browser_download_url":"http://` + r.Host + `/bin","digest":"` + digestOf(body) + `"}]}`))
		case "/bin":
			_, _ = w.Write([]byte(body))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), API: srv.URL, Repo: DefaultRepo}
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
	sib, err := os.ReadFile(filepath.Join(dir, "DiscordCli"))
	if err != nil {
		t.Fatal(err)
	}
	if string(sib) != body {
		t.Fatalf("sibling %q", sib)
	}
}

func TestApplyRejectsBadDigest(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "dis")
	if err := os.WriteFile(dest, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	wantName := AssetName(runtime.GOOS, runtime.GOARCH)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/Codezilla-jpg/plaincord/releases/latest":
			_, _ = w.Write([]byte(`{"tag_name":"v0.9.0","assets":[{"name":"` + wantName + `","browser_download_url":"http://` + r.Host + `/bin","digest":"` + digestOf("expected") + `"}]}`))
		case "/bin":
			_, _ = w.Write([]byte("tampered"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), API: srv.URL, Repo: DefaultRepo}
	if _, err := c.Apply("0.2.0", dest); !errors.Is(err, ErrBadDigest) {
		t.Fatalf("got %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "old" {
		t.Fatalf("replaced on bad digest: %q", got)
	}
}

func TestApplyRejectsForeignHost(t *testing.T) {
	wantName := AssetName(runtime.GOOS, runtime.GOARCH)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.9.0","assets":[{"name":"` + wantName + `","browser_download_url":"https://evil.example/bin","digest":"` + digestOf("x") + `"}]}`))
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), API: srv.URL, Repo: DefaultRepo}
	if _, err := c.Apply("0.2.0", filepath.Join(t.TempDir(), "dis")); !errors.Is(err, ErrBadURL) {
		t.Fatalf("got %v", err)
	}
}

func TestApplyRejectsMissingDigest(t *testing.T) {
	wantName := AssetName(runtime.GOOS, runtime.GOARCH)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.9.0","assets":[{"name":"` + wantName + `","browser_download_url":"http://` + r.Host + `/bin"}]}`))
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), API: srv.URL, Repo: DefaultRepo}
	if _, err := c.Apply("0.2.0", filepath.Join(t.TempDir(), "dis")); !errors.Is(err, ErrNoDigest) {
		t.Fatalf("got %v", err)
	}
}

func TestApplyNoUpdate(t *testing.T) {
	wantName := AssetName(runtime.GOOS, runtime.GOARCH)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","assets":[{"name":"` + wantName + `","browser_download_url":"http://example/bin"}]}`))
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), API: srv.URL, Repo: DefaultRepo}
	tag, err := c.Apply("0.2.0", filepath.Join(t.TempDir(), "dis"))
	if err != nil {
		t.Fatal(err)
	}
	if tag != "v0.2.0" {
		t.Fatalf("tag %s", tag)
	}
}
