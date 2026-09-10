# plaincord

A clean Discord terminal client. Servers, channel folders, text chat, and voice — nothing else.

Official **bot token** login only. No user accounts, no self-bots.

## Install

```bash
uv tool install git+https://github.com/Codezilla-jpg/plaincord
# or from a clone
uv sync
uv run plaincord --demo
```

Requires Python 3.11+.

## Login

1. Create an application at [Discord Developer Portal](https://discord.com/developers/applications).
2. Bot → Add Bot.
3. Privileged Gateway Intents → enable **Message Content Intent**.
4. Reset Token → copy it.

```bash
plaincord login
plaincord
```

Token goes to the OS keyring, or `~/.config/plaincord/token` (mode 0600). Override with `PLAINCORD_TOKEN`.

## Add a server

```bash
plaincord invite
```

Open the URL, pick a server. In the TUI press `a`, then `r` to reload.

Needed bot permissions: View Channel, Send Messages, Read Message History, Connect, Speak.

## Keys

| Key | Action |
|---|---|
| arrows / tab | move |
| enter | open text channel or join voice |
| r | reload chat (or server list) |
| j | join highlighted voice channel |
| l | leave call |
| m | mute / unmute |
| a | add server (invite URL) |
| esc | back to channel list |
| ctrl+q | quit |

Type in the composer and press enter to send.

Voice join puts the bot in the channel. Mute is Discord self-mute. No local mic/speaker routing.

## Not in scope

Server settings, roles, moderation, DMs, threads, forums, friend lists.

## Why a bot?

Discord forbids automated user accounts. plaincord is a bot client: it appears as your bot user in servers you invite it to.

## License

MIT
