# plaincord

Clean Discord TUI. Binary names: **dis** and **DiscordCli**.

Your **personal Discord account** — servers, channel folders, text chat, voice join/leave/mute.

Discord forbids unofficial user clients. Ban risk is yours.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Codezilla-jpg/plaincord/main/install.sh | sh
```

Puts `dis` and `DiscordCli` in `~/.local/bin`.

```bash
dis update
```

## Login

```bash
dis login
dis
```

Stores the account token in `~/.config/plaincord/token` (0600), or use `PLAINCORD_TOKEN`.

## Commands

| | |
|---|---|
| `dis` | start TUI |
| `dis --demo` | sample data |
| `dis login` | store account token |
| `dis logout` | remove token |
| `dis invite discord.gg/…` | join a server |
| `dis update` | update this binary |
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
