# plaincord

Clean Discord TUI. Binary names: **dc** and **DiscordCli**.

Servers, channel folders, text chat, voice join/leave/mute. Bot token only.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Codezilla-jpg/plaincord/main/install.sh | sh
```

Puts `dc` and `DiscordCli` in `~/.local/bin`.

```bash
dc update          # replace this install with the latest GitHub release
```

From source:

```bash
git clone https://github.com/Codezilla-jpg/plaincord.git
cd plaincord
go build -o dc ./cmd/dc
cp dc DiscordCli
```

## Login

1. [Discord Developer Portal](https://discord.com/developers/applications) → New App → Bot
2. Enable **Message Content Intent**
3. Copy the bot token

```bash
dc login
dc
# or
DiscordCli --demo
```

Token: `~/.config/plaincord/token` (0600) or `PLAINCORD_TOKEN`.

## Commands

| | |
|---|---|
| `dc` | start TUI |
| `dc --demo` | sample data |
| `dc login` | store token |
| `dc logout` | remove token |
| `dc invite` | bot invite URL |
| `dc update` | update this binary |
| `dc --version` | version |

## Keys

| Key | Action |
|---|---|
| arrows / tab | move |
| enter | open text channel or join voice |
| r | reload chat / servers |
| j | join highlighted voice channel |
| l | leave call |
| m | mute / unmute |
| a | add server (invite URL) |
| esc | back to channels |
| ctrl-c | quit |

Voice join puts the bot in the channel. Mute is Discord self-mute. No local mic/speaker routing.

## License

MIT
