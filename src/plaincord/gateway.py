from __future__ import annotations

import asyncio
from datetime import UTC, datetime
from typing import Protocol

from plaincord.auth import save_application_id
from plaincord.models import ChannelInfo, ChatMessage, GuildInfo, Kind


class GatewayListener(Protocol):
    def on_ready(self) -> None: ...
    def on_message(self, message: ChatMessage) -> None: ...
    def on_error(self, text: str) -> None: ...


class Gateway(Protocol):
    application_id: str | None

    async def start(self) -> None: ...
    async def close(self) -> None: ...
    async def guilds(self) -> list[GuildInfo]: ...
    async def channels(self, guild_id: int) -> list[ChannelInfo]: ...
    async def history(self, channel_id: int, limit: int = 80) -> list[ChatMessage]: ...
    async def send(self, channel_id: int, content: str) -> ChatMessage: ...
    async def join_voice(self, guild_id: int, channel_id: int, name: str) -> None: ...
    async def leave_voice(self) -> None: ...
    async def set_mute(self, muted: bool) -> None: ...


def _now() -> datetime:
    return datetime.now(UTC)


class FakeGateway:
    application_id = "0"

    def __init__(self, listener: GatewayListener) -> None:
        self.listener = listener
        self._guilds = [
            GuildInfo(1, "Home"),
            GuildInfo(2, "Friends"),
        ]
        self._channels: dict[int, list[ChannelInfo]] = {
            1: [
                ChannelInfo(10, "welcome", "text", None, 0),
                ChannelInfo(11, "Text", "category", None, 1),
                ChannelInfo(12, "general", "text", 11, 0),
                ChannelInfo(13, "random", "text", 11, 1),
                ChannelInfo(14, "Voice", "category", None, 2),
                ChannelInfo(15, "lounge", "voice", 14, 0),
                ChannelInfo(16, "gaming", "voice", 14, 1),
            ],
            2: [
                ChannelInfo(20, "chat", "text", None, 0),
                ChannelInfo(21, "call", "voice", None, 1),
            ],
        }
        self._history: dict[int, list[ChatMessage]] = {
            12: [
                ChatMessage(1, 12, "Ada", "hello", _now()),
                ChatMessage(2, 12, "Ben", "hi there", _now()),
            ],
            13: [ChatMessage(3, 13, "Ada", "off-topic lives here", _now())],
            10: [ChatMessage(4, 10, "System", "welcome to Home", _now())],
            20: [ChatMessage(5, 20, "Cara", "hey", _now())],
        }
        self._next_id = 100
        self._voice_channel_id: int | None = None
        self._voice_guild_id: int | None = None
        self._muted = False

    async def start(self) -> None:
        await asyncio.sleep(0)
        self.listener.on_ready()

    async def close(self) -> None:
        return

    async def guilds(self) -> list[GuildInfo]:
        return list(self._guilds)

    async def channels(self, guild_id: int) -> list[ChannelInfo]:
        return list(self._channels.get(guild_id, []))

    async def history(self, channel_id: int, limit: int = 80) -> list[ChatMessage]:
        return list(self._history.get(channel_id, [])[-limit:])

    async def send(self, channel_id: int, content: str) -> ChatMessage:
        self._next_id += 1
        message = ChatMessage(self._next_id, channel_id, "you", content, _now())
        self._history.setdefault(channel_id, []).append(message)
        self.listener.on_message(message)
        return message

    async def join_voice(self, guild_id: int, channel_id: int, name: str) -> None:
        del name
        self._voice_guild_id = guild_id
        self._voice_channel_id = channel_id
        self._muted = False

    async def leave_voice(self) -> None:
        self._voice_channel_id = None
        self._voice_guild_id = None
        self._muted = False

    async def set_mute(self, muted: bool) -> None:
        if self._voice_channel_id is None:
            raise RuntimeError("not in a call")
        self._muted = muted


def _kind_for(channel) -> Kind | None:
    import discord

    if isinstance(channel, discord.CategoryChannel):
        return "category"
    if isinstance(channel, discord.VoiceChannel):
        return "voice"
    if isinstance(channel, discord.TextChannel):
        return "text"
    return None


def _to_chat(message) -> ChatMessage:
    author = message.author.display_name if message.author else "?"
    stamp = message.created_at
    if stamp.tzinfo is None:
        stamp = stamp.replace(tzinfo=UTC)
    return ChatMessage(
        id=int(message.id),
        channel_id=int(message.channel.id),
        author=author,
        content=message.content or "",
        timestamp=stamp,
    )


class DiscordGateway:
    def __init__(self, token: str, listener: GatewayListener) -> None:
        self.token = token
        self.listener = listener
        self.application_id: str | None = None
        self._client = None
        self._voice = None

    def _build_client(self):
        import discord

        intents = discord.Intents.default()
        intents.message_content = True
        intents.guilds = True
        intents.voice_states = True
        client = discord.Client(intents=intents)

        @client.event
        async def on_ready() -> None:
            app = getattr(client, "application", None)
            if app is not None:
                self.application_id = str(app.id)
                save_application_id(self.application_id)
            elif client.user is not None:
                self.application_id = str(client.user.id)
                save_application_id(self.application_id)
            self.listener.on_ready()

        @client.event
        async def on_message(message) -> None:
            if message.guild is None:
                return
            self.listener.on_message(_to_chat(message))

        return client

    async def start(self) -> None:
        import discord

        self._client = self._build_client()
        try:
            await self._client.start(self.token)
        except discord.LoginFailure:
            self.listener.on_error("Login failed — token rejected")
            raise

    async def close(self) -> None:
        if self._voice is not None:
            try:
                await self._voice.disconnect(force=True)
            except Exception:
                pass
            self._voice = None
        if self._client is not None:
            await self._client.close()

    async def guilds(self) -> list[GuildInfo]:
        if self._client is None:
            return []
        guilds = sorted(self._client.guilds, key=lambda g: g.name.lower())
        return [GuildInfo(int(g.id), g.name) for g in guilds]

    async def channels(self, guild_id: int) -> list[ChannelInfo]:
        if self._client is None:
            return []
        guild = self._client.get_guild(guild_id)
        if guild is None:
            return []
        result: list[ChannelInfo] = []
        for channel in guild.channels:
            kind = _kind_for(channel)
            if kind is None:
                continue
            category_id = None
            if kind != "category":
                category = getattr(channel, "category", None)
                category_id = int(category.id) if category is not None else None
            result.append(
                ChannelInfo(
                    id=int(channel.id),
                    name=channel.name,
                    kind=kind,
                    category_id=category_id,
                    position=int(getattr(channel, "position", 0)),
                )
            )
        return result

    async def history(self, channel_id: int, limit: int = 80) -> list[ChatMessage]:
        if self._client is None:
            return []
        channel = self._client.get_channel(channel_id)
        if channel is None or not hasattr(channel, "history"):
            return []
        messages = [msg async for msg in channel.history(limit=limit)]
        messages.reverse()
        return [_to_chat(msg) for msg in messages]

    async def send(self, channel_id: int, content: str) -> ChatMessage:
        if self._client is None:
            raise RuntimeError("not connected")
        channel = self._client.get_channel(channel_id)
        if channel is None or not hasattr(channel, "send"):
            raise RuntimeError("channel not found")
        sent = await channel.send(content)
        return _to_chat(sent)

    async def join_voice(self, guild_id: int, channel_id: int, name: str) -> None:
        del name
        import discord

        if self._client is None:
            raise RuntimeError("not connected")
        channel = self._client.get_channel(channel_id)
        if not isinstance(channel, discord.VoiceChannel):
            raise RuntimeError("not a voice channel")
        if int(channel.guild.id) != guild_id:
            raise RuntimeError("voice channel is on another server")
        if self._voice is not None and self._voice.is_connected():
            await self._voice.move_to(channel)
            return
        self._voice = await channel.connect()

    async def leave_voice(self) -> None:
        if self._voice is not None:
            await self._voice.disconnect()
            self._voice = None

    async def set_mute(self, muted: bool) -> None:
        if self._client is None or self._voice is None or not self._voice.is_connected():
            raise RuntimeError("not in a call")
        channel = self._voice.channel
        await channel.guild.change_voice_state(channel=channel, self_mute=muted, self_deaf=False)
