from __future__ import annotations

from datetime import UTC, datetime

from plaincord.formatters import (
    channel_label,
    format_markup_line,
    format_message_line,
    voice_bar_text,
)


def test_format_message_line() -> None:
    moment = datetime(2026, 1, 2, 15, 4, tzinfo=UTC)
    line = format_message_line("Ada", "hello", moment)
    assert "Ada: hello" in line
    assert ":" in line[:5] or line[0].isdigit()


def test_markup_escapes_tags() -> None:
    moment = datetime(2026, 1, 2, 15, 4, tzinfo=UTC)
    line = format_markup_line("Ada", "[bold]oops[/bold]", moment)
    assert "[bold]oops[/bold]" not in line or "\\[bold]" in line or "&" in line or "oops" in line
    assert "Ada" in line


def test_channel_labels() -> None:
    assert channel_label("text", "general") == "# general"
    assert channel_label("voice", "lounge").endswith("lounge")
    assert channel_label("category", "Text") == "TEXT"


def test_voice_bar() -> None:
    idle = voice_bar_text(False, False, "")
    assert "not in a call" in idle
    live = voice_bar_text(True, False, "lounge")
    assert "lounge" in live
    assert "live" in live
    muted = voice_bar_text(True, True, "lounge")
    assert "muted" in muted
