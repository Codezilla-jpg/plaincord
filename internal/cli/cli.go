package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/Codezilla-jpg/plaincord/internal/auth"
	"github.com/Codezilla-jpg/plaincord/internal/chantree"
	"github.com/Codezilla-jpg/plaincord/internal/ui"
	"github.com/Codezilla-jpg/plaincord/internal/update"
)

var Version = "0.2.0"

func Run(args []string, stdin *os.File, stdout, stderr io.Writer) int {
	name := "dc"
	if len(args) > 0 {
		name = filepath.Base(args[0])
	}
	rest := []string{}
	if len(args) > 1 {
		rest = args[1:]
	}
	demo := false
	filtered := make([]string, 0, len(rest))
	for _, arg := range rest {
		switch arg {
		case "--demo":
			demo = true
		case "-h", "--help", "help":
			fmt.Fprint(stdout, helpText(name))
			return 0
		case "-v", "--version", "version":
			fmt.Fprintf(stdout, "%s %s\n", name, Version)
			return 0
		default:
			filtered = append(filtered, arg)
		}
	}
	cmd := ""
	if len(filtered) > 0 {
		cmd = filtered[0]
	}
	switch cmd {
	case "":
		if err := ui.Run(demo, ""); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	case "login":
		return cmdLogin(stdin, stdout, stderr)
	case "logout":
		if err := auth.DeleteToken(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, "Token removed.")
		return 0
	case "invite":
		id, err := auth.LoadApplicationID()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if id == "" {
			fmt.Fprintln(stderr, "No bot id stored yet. Run dc once, then retry.")
			return 1
		}
		fmt.Fprintln(stdout, chantree.InviteURL(id))
		return 0
	case "update":
		c := update.New()
		tag, err := c.Apply(Version, "")
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if !update.NeedsUpdate(Version, tag) {
			fmt.Fprintf(stdout, "already up to date (%s)\n", update.Normalize(tag))
			return 0
		}
		fmt.Fprintf(stdout, "updated %s -> %s\n", Version, update.Normalize(tag))
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", cmd)
		fmt.Fprint(stderr, helpText(name))
		return 2
	}
}

func cmdLogin(stdin *os.File, stdout, stderr io.Writer) int {
	fmt.Fprintln(stdout, "Create a bot: https://discord.com/developers/applications")
	fmt.Fprintln(stdout, "Enable Privileged Gateway Intent: Message Content.")
	fmt.Fprint(stdout, "Bot token: ")
	var token string
	if stdin != nil && term.IsTerminal(int(stdin.Fd())) {
		pw, err := term.ReadPassword(int(stdin.Fd()))
		fmt.Fprintln(stdout)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		token = string(pw)
	} else {
		buf, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		token = strings.TrimSpace(string(buf))
	}
	if err := auth.SaveToken(token); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, "Saved. Start with: dc")
	return 0
}

func helpText(name string) string {
	return fmt.Sprintf(`%s — clean Discord TUI (also installed as DiscordCli)

Usage:
  %s                 start the client
  %s --demo          sample data, no token
  %s login           store a bot token
  %s logout          remove the stored token
  %s invite          print the bot invite URL
  %s update          replace this binary with the latest GitHub release
  %s --version

Keys: arrows  enter  r reload  j join  l leave  m mute  a add  ctrl-c quit
`, name, name, name, name, name, name, name, name)
}
