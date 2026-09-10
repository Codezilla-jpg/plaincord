from __future__ import annotations

from plaincord.gateway import FakeGateway


class _Sink:
    def __init__(self) -> None:
        self.ready = False
        self.messages = []
        self.errors = []

    def on_ready(self) -> None:
        self.ready = True

    def on_message(self, message) -> None:
        self.messages.append(message)

    def on_error(self, text: str) -> None:
        self.errors.append(text)


async def test_fake_gateway_flow() -> None:
    sink = _Sink()
    gw = FakeGateway(sink)
    await gw.start()
    assert sink.ready is True
    guilds = await gw.guilds()
    assert [g.name for g in guilds] == ["Home", "Friends"]
    channels = await gw.channels(1)
    assert any(c.name == "general" and c.kind == "text" for c in channels)
    assert any(c.name == "lounge" and c.kind == "voice" for c in channels)
    history = await gw.history(12)
    assert history
    sent = await gw.send(12, "ping")
    assert sent.content == "ping"
    assert sink.messages[-1].content == "ping"
    await gw.join_voice(1, 15, "lounge")
    await gw.set_mute(True)
    await gw.leave_voice()
    await gw.close()
