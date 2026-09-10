from __future__ import annotations

import pytest

from plaincord.voice import VoiceController


def test_join_mute_leave() -> None:
    voice = VoiceController()
    assert voice.state.connected is False
    voice.on_joined(1, 15, "lounge")
    assert voice.state.connected is True
    assert voice.state.channel_name == "lounge"
    assert voice.state.muted is False
    assert voice.toggle_mute() is True
    assert voice.state.muted is True
    assert voice.toggle_mute() is False
    voice.on_left()
    assert voice.state.connected is False
    assert voice.state.channel_name == ""


def test_mute_without_call_raises() -> None:
    voice = VoiceController()
    with pytest.raises(RuntimeError):
        voice.toggle_mute()
