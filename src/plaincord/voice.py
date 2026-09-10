from __future__ import annotations

from plaincord.models import VoiceState


class VoiceController:
    def __init__(self) -> None:
        self.state = VoiceState()

    def on_joined(self, guild_id: int, channel_id: int, name: str) -> VoiceState:
        self.state = VoiceState(
            connected=True,
            muted=False,
            guild_id=guild_id,
            channel_id=channel_id,
            channel_name=name,
        )
        return self.state

    def on_left(self) -> VoiceState:
        self.state = VoiceState()
        return self.state

    def toggle_mute(self) -> bool:
        if not self.state.connected:
            raise RuntimeError("not in a call")
        self.state.muted = not self.state.muted
        return self.state.muted

    def set_mute(self, muted: bool) -> bool:
        if not self.state.connected:
            raise RuntimeError("not in a call")
        self.state.muted = bool(muted)
        return self.state.muted
