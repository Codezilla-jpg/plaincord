package invite

import (
	"errors"
	"net/url"
	"strings"
)

var ErrInvalid = errors.New("invalid invite")

func Parse(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", ErrInvalid
	}
	s = strings.Trim(s, "/")
	if strings.Contains(s, "://") || strings.HasPrefix(s, "discord.gg/") || strings.HasPrefix(s, "discord.com/") || strings.HasPrefix(s, "www.discord.") {
		if !strings.Contains(s, "://") {
			s = "https://" + s
		}
		u, err := url.Parse(s)
		if err != nil {
			return "", ErrInvalid
		}
		host := strings.ToLower(u.Host)
		switch {
		case host == "discord.gg" || strings.HasSuffix(host, ".discord.gg"):
			s = strings.Trim(u.Path, "/")
		case host == "discord.com" || host == "www.discord.com" || host == "discordapp.com":
			s = strings.Trim(u.Path, "/")
			s = strings.TrimPrefix(s, "invite/")
		default:
			return "", ErrInvalid
		}
	}
	s = strings.Trim(s, "/")
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	if s == "" || strings.ContainsAny(s, "/ \t") {
		return "", ErrInvalid
	}
	return s, nil
}
