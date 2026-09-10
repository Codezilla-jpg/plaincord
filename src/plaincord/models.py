from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import Literal

Kind = Literal["category", "text", "voice"]


@dataclass(frozen=True)
class GuildInfo:
    id: int
    name: str


@dataclass(frozen=True)
class ChannelInfo:
    id: int
    name: str
    kind: Kind
    category_id: int | None
    position: int


@dataclass(frozen=True)
class ChatMessage:
    id: int
    channel_id: int
    author: str
    content: str
    timestamp: datetime


@dataclass
class VoiceState:
    connected: bool = False
    muted: bool = False
    guild_id: int | None = None
    channel_id: int | None = None
    channel_name: str = ""


@dataclass(frozen=True)
class TreeNode:
    id: int | None
    name: str
    kind: Kind
    children: tuple[TreeNode, ...] = ()
    channel: ChannelInfo | None = None
