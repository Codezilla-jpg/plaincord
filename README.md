# plaincord

Clean Discord TUI. Commands: **dis** and **DiscordCli**.

Personal Discord account — servers, channel folders, text chat, voice join/leave/mute.

Discord forbids unofficial user clients. Ban risk is yours.

## Install

macOS (Intel + Apple Silicon) and Linux:

```bash
curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/Codezilla-jpg/plaincord/main/install.sh | sh
```

Installs `dis` and `DiscordCli` to `~/.local/bin`. If `dis` is not found:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

Windows: download `dis_windows_amd64.exe` from [Releases](https://github.com/Codezilla-jpg/plaincord/releases/latest).

Binaries: `dis_darwin_arm64` (Apple Silicon), `dis_darwin_amd64` (Intel Mac), `dis_linux_amd64`, `dis_linux_arm64`, `dis_windows_amd64.exe`.

From source:

```bash
git clone https://github.com/Codezilla-jpg/plaincord.git
cd plaincord
go build -o dis ./cmd/dis
```

## Update / uninstall

```bash
dis update
dis uninstall
```

`uninstall` removes `dis`, `DiscordCli`, leftover `dc`, and `~/.config/plaincord`.

## Login

```bash
dis login
dis
```

Stores the account token in the OS keyring (macOS Keychain / Secret Service / Windows Credential Manager). File fallback: `~/.config/plaincord/token` (0600). `PLAINCORD_TOKEN` overrides for the process only — do not export it long-term.

## Commands

| | |
|---|---|
| `dis` | start TUI |
| `dis --demo` | sample data |
| `dis login` | store account token |
| `dis logout` | remove token |
| `dis invite discord.gg/…` | join a server |
| `dis update` | update this binary |
| `dis uninstall` | remove install + token |
| `dis --version` | version |

## Keys

| Key | Action |
|---|---|
| enter / → | open (server → channels → chat or join voice) |
| esc / ← | back |
| arrows | move |
| r | reload this level |
| l | leave call |
| m | mute / unmute |
| a | join server (paste invite) |
| ctrl-c | quit |

Voice join puts your account in the channel. Mute is Discord self-mute. No local mic/speaker routing.

## License

MIT
