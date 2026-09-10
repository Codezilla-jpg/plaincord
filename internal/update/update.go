package update

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
)

type Client struct {
	HTTP *http.Client
	API  string
	Repo string
}

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
}

func New() *Client {
	c := &Client{
		API:  "https://api.github.com",
		Repo: DefaultRepo,
	}
	c.HTTP = &http.Client{
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
	return c
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

func (c *Client) Latest(current string) (Release, Asset, error) {
	c.ensureHTTP()
	if c.API == "" {
		c.API = "https://api.github.com"
	}
	if c.Repo == "" {
		c.Repo = DefaultRepo
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(c.API, "/")+"/repos/"+c.Repo+"/releases/latest", nil)
	if err != nil {
		return Release{}, Asset{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "plaincord/"+current)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Release{}, Asset{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return Release{}, Asset{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Release{}, Asset{}, fmt.Errorf("github release: %s", resp.Status)
	}
	var rel Release
	if err := json.Unmarshal(body, &rel); err != nil {
		return Release{}, Asset{}, err
	}
	want := AssetName(runtime.GOOS, runtime.GOARCH)
	for _, asset := range rel.Assets {
		if asset.Name == want {
			return rel, asset, nil
		}
	}
	return rel, Asset{}, fmt.Errorf("no asset %s in %s", want, rel.TagName)
}

func (c *Client) Apply(current, dest string) (string, error) {
	rel, asset, err := c.Latest(current)
	if err != nil {
		return "", err
	}
	if !NeedsUpdate(current, rel.TagName) {
		return rel.TagName, nil
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
	if err := c.replace(asset, dest); err != nil {
		return "", err
	}
	c.syncSibling(dest)
	return rel.TagName, nil
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
	req.Header.Set("User-Agent", "plaincord")
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
	sum := h.Sum(nil)
	if subtle.ConstantTimeCompare(sum, want) != 1 {
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

func (c *Client) allowedURL(u *url.URL) bool {
	if u == nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	if apiHost := c.apiHostname(); apiHost != "" && host == apiHost {
		if isLoopback(host) {
			return u.Scheme == "http" || u.Scheme == "https"
		}
		return u.Scheme == "https"
	}
	if u.Scheme != "https" {
		return false
	}
	switch host {
	case "github.com", "api.github.com",
		"objects.githubusercontent.com",
		"release-assets.githubusercontent.com",
		"github-releases.githubusercontent.com":
		return true
	}
	return strings.HasSuffix(host, ".githubusercontent.com")
}

func (c *Client) apiHostname() string {
	u, err := url.Parse(c.API)
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
	c.HTTP = &http.Client{
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
