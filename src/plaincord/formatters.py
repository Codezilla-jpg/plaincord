from __future__ import annotations

from datetime import datetime

from rich.markup import escape


def format_clock(moment: datetime) -> str:
    local = moment.astimezone() if moment.tzinfo else moment
    return local.strftime("%H:%M")


def format_message_line(author: str, content: str, moment: datetime) -> str:
    body = content.replace("\r\n", "\n").strip("\n")
    if not body:
        body = "·"
    return f"{format_clock(moment)} {author}: {body}"


def format_markup_line(author: str, content: str, moment: datetime) -> str:
    body = escape(content.replace("\r\n", "\n").strip("\n") or "·")
    return f"[dim]{escape(format_clock(moment))}[/] [b]{escape(author)}[/]  {body}"


def channel_label(kind: str, name: str) -> str:
    if kind == "voice":
        return f"🔊 {name}"
    if kind == "category":
        return name.upper()
    return f"# {name}"


def voice_bar_text(connected: bool, muted: bool, channel_name: str) -> str:
    if not connected:
        return "Voice  ·  not in a call    j join   l leave   m mute"
    state = "muted" if muted else "live"
    return f"Voice  ·  {channel_name}  ·  {state}    m mute   l leave"
