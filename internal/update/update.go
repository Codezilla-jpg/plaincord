package update

import (
	"bufio"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	DefaultRepo = "Codezilla-jpg/plaincord"
	maxAsset    = 40 << 20
)

var (
	ErrNoDigest     = errors.New("release asset has no sha256 digest")
	ErrBadDigest    = errors.New("sha256 mismatch")
	ErrBadURL       = errors.New("download host not allowed")
	ErrTooLarge     = errors.New("asset exceeds size limit")
	ErrTooManyRedir = errors.New("too many redirects")
	tagRe           = regexp.MustCompile(`/releases/download/(v[^/]+)/`)
)

type Client struct {
	HTTP *http.Client
	Base string
	Repo string
}

func New() *Client {
	c := &Client{
		Base: "https://github.com",
		Repo: DefaultRepo,
	}
	c.HTTP = c.newHTTP()
	return c
}

func (c *Client) newHTTP() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return ErrTooManyRedir
			}
			if !c.allowedURL(req.URL) {
				return fmt.Errorf("%w: %s", ErrBadURL, req.URL.Host)
			}
			return nil
		},
	}
}

func AssetName(goos, goarch string) string {
	name := fmt.Sprintf("dis_%s_%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

func Normalize(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func NeedsUpdate(current, latest string) bool {
	c, l := Normalize(current), Normalize(latest)
	return l != "" && c != l
}

func ParseTag(raw string) string {
	m := tagRe.FindStringSubmatch(raw)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

func ParseSums(body, name string) (string, error) {
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[len(fields)-1] == name {
			return "sha256:" + fields[0], nil
		}
	}
	return "", ErrNoDigest
}

func (c *Client) fileURL(name string) string {
	base := c.Base
	if base == "" {
		base = "https://github.com"
	}
	repo := c.Repo
	if repo == "" {
		repo = DefaultRepo
	}
	return strings.TrimRight(base, "/") + "/" + repo + "/releases/latest/download/" + name
}

func (c *Client) Latest(current string) (tag string, assetURL, digest string, err error) {
	c.ensureHTTP()
	name := AssetName(runtime.GOOS, runtime.GOARCH)
	assetURL = c.fileURL(name)
	tag, err = c.peekTag(assetURL)
	if err != nil {
		return "", "", "", err
	}
	sums, err := c.get(c.fileURL("SHA256SUMS"), 1<<20)
	if err != nil {
		return "", "", "", err
	}
	digest, err = ParseSums(string(sums), name)
	if err != nil {
		return "", "", "", err
	}
	_ = current
	return tag, assetURL, digest, nil
}

func (c *Client) Apply(current, dest string) (string, error) {
	tag, assetURL, digest, err := c.Latest(current)
	if err != nil {
		return "", err
	}
	if !NeedsUpdate(current, tag) {
		return tag, nil
	}
	if dest == "" {
		dest, err = os.Executable()
		if err != nil {
			return "", err
		}
		dest, err = filepath.EvalSymlinks(dest)
		if err != nil {
			return "", err
		}
	}
	asset := Asset{Name: AssetName(runtime.GOOS, runtime.GOARCH), URL: assetURL, Digest: digest}
	if err := c.replace(asset, dest); err != nil {
		return "", err
	}
	c.syncSibling(dest)
	return tag, nil
}

type Asset struct {
	Name   string
	URL    string
	Digest string
}

func (c *Client) peekTag(rawURL string) (string, error) {
	req, err := http.NewRequest(http.MethodHead, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "dis")
	client := c.newHTTP()
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	loc := resp.Header.Get("Location")
	if loc == "" {
		loc = rawURL
	} else if u, err := url.Parse(rawURL); err == nil {
		if ref, err := u.Parse(loc); err == nil {
			loc = ref.String()
		}
	}
	if !c.allowedURLMust(loc) {
		return "", fmt.Errorf("%w: %s", ErrBadURL, loc)
	}
	tag := ParseTag(loc)
	if tag == "" {
		return "", fmt.Errorf("could not resolve latest tag")
	}
	return tag, nil
}

func (c *Client) get(rawURL string, limit int64) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "dis")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, limit))
}

func (c *Client) replace(asset Asset, dest string) error {
	c.ensureHTTP()
	want, err := parseDigest(asset.Digest)
	if err != nil {
		return err
	}
	u, err := url.Parse(asset.URL)
	if err != nil {
		return err
	}
	if !c.allowedURL(u) {
		return fmt.Errorf("%w: %s", ErrBadURL, u.Host)
	}
	req, err := http.NewRequest(http.MethodGet, asset.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "dis")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: %s", resp.Status)
	}
	if resp.ContentLength > maxAsset {
		return ErrTooLarge
	}
	tmp := dest + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, copyErr := io.Copy(out, io.TeeReader(io.LimitReader(resp.Body, maxAsset+1), h))
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n > maxAsset {
		_ = os.Remove(tmp)
		return ErrTooLarge
	}
	if subtle.ConstantTimeCompare(h.Sum(nil), want) != 1 {
		_ = os.Remove(tmp)
		return ErrBadDigest
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func parseDigest(d string) ([]byte, error) {
	d = strings.TrimSpace(strings.ToLower(d))
	if !strings.HasPrefix(d, "sha256:") {
		return nil, ErrNoDigest
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(d, "sha256:"))
	if err != nil || len(raw) != sha256.Size {
		return nil, ErrNoDigest
	}
	return raw, nil
}

func (c *Client) allowedURLMust(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return c.allowedURL(u)
}

func (c *Client) allowedURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	if baseHost := c.baseHostname(); baseHost != "" && host == baseHost {
		if isLoopback(host) {
			return u.Scheme == "http" || u.Scheme == "https"
		}
		return u.Scheme == "https"
	}
	if u.Scheme != "https" {
		return false
	}
	switch host {
	case "github.com",
		"objects.githubusercontent.com",
		"release-assets.githubusercontent.com",
		"github-releases.githubusercontent.com":
		return true
	}
	return strings.HasSuffix(host, ".githubusercontent.com")
}

func (c *Client) baseHostname() string {
	u, err := url.Parse(c.Base)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c *Client) ensureHTTP() {
	if c.HTTP != nil {
		return
	}
	c.HTTP = c.newHTTP()
}

func (c *Client) syncSibling(dest string) {
	base := filepath.Base(dest)
	dir := filepath.Dir(dest)
	var other string
	switch base {
	case "dis", "dis.exe":
		other = "DiscordCli"
	case "DiscordCli", "DiscordCli.exe":
		other = "dis"
	default:
		return
	}
	if runtime.GOOS == "windows" {
		other += ".exe"
	}
	path := filepath.Join(dir, other)
	data, err := os.ReadFile(dest)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o755)
}
