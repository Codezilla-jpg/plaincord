from __future__ import annotations

from plaincord.app import PlaincordApp


async def test_demo_loads_servers_and_reload() -> None:
    app = PlaincordApp(demo=True)
    async with app.run_test(size=(140, 40)) as pilot:
        for _ in range(30):
            await pilot.pause(0.05)
            if app.current_guild is not None:
                break
        assert app.current_guild is not None
        assert app.current_guild.name == "Home"
        listing = app.query_one("#guilds")
        assert len(list(listing.children)) == 2
        await app.action_reload()
        await pilot.pause()
        bar = str(app.query_one("#voicebar").render())
        assert "not in a call" in bar
