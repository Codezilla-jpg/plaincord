package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const DefaultRepo = "Codezilla-jpg/plaincord"

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
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

func New() *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 60 * time.Second},
		API:  "https://api.github.com",
		Repo: DefaultRepo,
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

func (c *Client) Latest(current string) (Release, Asset, error) {
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}
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
	body, err := io.ReadAll(resp.Body)
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
	if err := c.replace(asset.URL, dest); err != nil {
		return "", err
	}
	c.syncSibling(dest)
	return rel.TagName, nil
}

func (c *Client) replace(url, dest string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
	tmp := dest + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
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
