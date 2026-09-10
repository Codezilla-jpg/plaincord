# plaincord

Clean Discord TUI. Commands: **dis** and **DiscordCli**.

Personal Discord account — servers, channel folders, text chat, voice join/leave/mute.

Discord forbids unofficial user clients. Ban risk is yours.

## Install

macOS (Intel + Apple Silicon) and Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/Codezilla-jpg/plaincord/main/install.sh | sh
```

Installs `dis` and `DiscordCli` to `~/.local/bin`. If `dis` is not found:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

Windows: download `dis_windows_amd64.exe` from [Releases](https://github.com/Codezilla-jpg/plaincord/releases/latest).

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

Token: `~/.config/plaincord/token` (0600) or `PLAINCORD_TOKEN`.

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
| arrows / tab | move |
| enter | open text channel or join voice |
| r | reload chat / servers |
| j | join highlighted voice channel |
| l | leave call |
| m | mute / unmute |
| a | join server (paste invite) |
| esc | back to channels |
| ctrl-c | quit |

Voice join puts your account in the channel. Mute is Discord self-mute. No local mic/speaker routing.

## License

MIT
