# plaincord

Clean Discord TUI. Binary names: **dc** and **DiscordCli**.

Your **personal Discord account** — servers, channel folders, text chat, voice join/leave/mute.

Discord forbids unofficial user clients. Ban risk is yours.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Codezilla-jpg/plaincord/main/install.sh | sh
```

Puts `dc` and `DiscordCli` in `~/.local/bin`.

```bash
dc update
```

## Login

```bash
dc login
dc
```

Stores the account token in `~/.config/plaincord/token` (0600), or use `PLAINCORD_TOKEN`.

## Commands

| | |
|---|---|
| `dc` | start TUI |
| `dc --demo` | sample data |
| `dc login` | store account token |
| `dc logout` | remove token |
| `dc invite discord.gg/…` | join a server |
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
| a | join server (paste invite) |
| esc | back to channels |
| ctrl-c | quit |

Voice join puts your account in the channel. Mute is Discord self-mute. No local mic/speaker routing.

## License

MIT
