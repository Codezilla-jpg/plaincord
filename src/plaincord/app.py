from __future__ import annotations

import asyncio
from pathlib import Path

from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Horizontal, Vertical
from textual.message import Message
from textual.screen import ModalScreen, Screen
from textual.widgets import (
    Button,
    Footer,
    Header,
    Input,
    Label,
    ListItem,
    ListView,
    RichLog,
    Static,
    Tree,
)

from plaincord.auth import load_application_id, load_token, looks_like_token, save_token
from plaincord.formatters import channel_label, format_markup_line, voice_bar_text
from plaincord.gateway import DiscordGateway, FakeGateway, Gateway
from plaincord.models import ChannelInfo, ChatMessage, GuildInfo, TreeNode
from plaincord.tree import build_channel_tree, invite_url
from plaincord.voice import VoiceController


class ReadyMsg(Message):
    pass


class ChatMsg(Message):
    def __init__(self, message: ChatMessage) -> None:
        super().__init__()
        self.message = message


class ErrorMsg(Message):
    def __init__(self, text: str) -> None:
        super().__init__()
        self.text = text


class GuildItem(ListItem):
    def __init__(self, guild: GuildInfo) -> None:
        super().__init__(Label(guild.name, markup=False))
        self.guild = guild


class _Listener:
    def __init__(self, app: PlaincordApp) -> None:
        self.app = app

    def on_ready(self) -> None:
        self.app.post_message(ReadyMsg())

    def on_message(self, message: ChatMessage) -> None:
        self.app.post_message(ChatMsg(message))

    def on_error(self, text: str) -> None:
        self.app.post_message(ErrorMsg(text))


class LoginScreen(Screen):
    BINDINGS = [Binding("ctrl+q", "app.quit", "Quit")]

    def compose(self) -> ComposeResult:
        with Vertical(id="login-box"):
            yield Label("plaincord", id="login-title")
            yield Label(
                "Paste a Discord bot token. Create one at discord.com/developers.",
                id="login-hint",
            )
            yield Input(placeholder="Bot token", password=True, id="token")

    def on_input_submitted(self, event: Input.Submitted) -> None:
        token = event.value.strip()
        if not looks_like_token(token):
            self.notify("Token looks invalid", severity="error")
            return
        save_token(token)
        app = self.app
        assert isinstance(app, PlaincordApp)
        app.token = token
        self.dismiss()
        app.start_gateway()


class InviteModal(ModalScreen[None]):
    BINDINGS = [
        Binding("escape", "close", "Close"),
        Binding("enter", "close", "Close"),
    ]

    def __init__(self, url: str, hint: str) -> None:
        super().__init__()
        self.url = url
        self.hint = hint

    def compose(self) -> ComposeResult:
        with Vertical(id="invite"):
            yield Label("Add a server")
            yield Static(self.hint, id="invite-hint")
            yield Static(self.url, id="invite-url")
            yield Button("Close", id="close", variant="primary")

    def on_button_pressed(self, event: Button.Pressed) -> None:
        del event
        self.dismiss()

    def action_close(self) -> None:
        self.dismiss()


class PlaincordApp(App):
    TITLE = "plaincord"
    CSS_PATH = Path(__file__).with_name("app.tcss")
    BINDINGS = [
        Binding("ctrl+q", "quit", "Quit"),
        Binding("r", "reload", "Reload"),
        Binding("m", "mute", "Mute"),
        Binding("j", "join_voice", "Join"),
        Binding("l", "leave_voice", "Leave"),
        Binding("a", "add_server", "Add"),
        Binding("escape", "focus_nav", "Nav", show=False),
    ]

    def __init__(self, *, demo: bool = False, token: str | None = None) -> None:
        super().__init__()
        self.demo = demo
        self.token = token
        self.gateway: Gateway | None = None
        self.gw_task: asyncio.Task | None = None
        self.voice = VoiceController()
        self.current_guild: GuildInfo | None = None
        self.current_channel: ChannelInfo | None = None
        self.seen_ids: set[int] = set()
        self._tree_nodes: dict[int, TreeNode] = {}

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        with Horizontal(id="body"):
            yield ListView(id="guilds")
            yield Tree("channels", id="channels")
            with Vertical(id="main"):
                yield Static("select a channel", id="chat-title")
                yield RichLog(id="chat", highlight=False, markup=True, wrap=True)
                yield Input(placeholder="Message…", id="composer")
        yield Static(voice_bar_text(False, False, ""), id="voicebar")
        yield Footer()

    def on_mount(self) -> None:
        if self.demo:
            self.start_gateway()
            return
        token = self.token or load_token()
        if not token:
            self.push_screen(LoginScreen())
            return
        self.token = token
        self.start_gateway()

    def start_gateway(self) -> None:
        listener = _Listener(self)
        if self.demo:
            self.gateway = FakeGateway(listener)
        else:
            if not self.token:
                self.push_screen(LoginScreen())
                return
            self.gateway = DiscordGateway(self.token, listener)
        self.gw_task = asyncio.create_task(self._run_gateway())
        self.query_one("#chat-title", Static).update("connecting…")

    async def _run_gateway(self) -> None:
        assert self.gateway is not None
        try:
            await self.gateway.start()
        except asyncio.CancelledError:
            raise
        except Exception as exc:
            self.post_message(ErrorMsg(str(exc) or exc.__class__.__name__))

    async def on_unmount(self) -> None:
        if self.gw_task is not None:
            self.gw_task.cancel()
        if self.gateway is not None:
            try:
                await self.gateway.close()
            except Exception:
                pass

    async def on_ready_msg(self, event: ReadyMsg) -> None:
        del event
        await self.refresh_guilds()
        self.query_one("#chat-title", Static).update("select a channel")
        self.notify("connected")

    def on_error_msg(self, event: ErrorMsg) -> None:
        self.notify(event.text, severity="error")
        self.query_one("#chat-title", Static).update(event.text)

    def on_chat_msg(self, event: ChatMsg) -> None:
        message = event.message
        if message.id in self.seen_ids:
            return
        if self.current_channel is None or message.channel_id != self.current_channel.id:
            return
        self._write_message(message)

    async def on_list_view_selected(self, event: ListView.Selected) -> None:
        item = event.item
        if not isinstance(item, GuildItem):
            return
        await self.open_guild(item.guild)

    async def on_tree_node_selected(self, event: Tree.NodeSelected) -> None:
        data = event.node.data
        if not isinstance(data, TreeNode) or data.channel is None:
            return
        if data.kind == "text":
            await self.open_text(data.channel)
        elif data.kind == "voice":
            await self.join_channel(data.channel)

    async def on_input_submitted(self, event: Input.Submitted) -> None:
        if event.input.id != "composer":
            return
        text = event.value.strip()
        event.input.value = ""
        if not text or self.gateway is None or self.current_channel is None:
            return
        if self.current_channel.kind != "text":
            self.notify("select a text channel", severity="warning")
            return
        try:
            sent = await self.gateway.send(self.current_channel.id, text)
        except Exception as exc:
            self.notify(str(exc), severity="error")
            return
        self._write_message(sent)

    async def open_guild(self, guild: GuildInfo) -> None:
        if self.gateway is None:
            return
        self.current_guild = guild
        channels = await self.gateway.channels(guild.id)
        tree = self.query_one("#channels", Tree)
        tree.clear()
        tree.root.label = guild.name
        tree.root.expand()
        self._tree_nodes.clear()
        for node in build_channel_tree(channels):
            self._add_tree_node(tree.root, node)
        self.current_channel = None
        self.query_one("#chat-title", Static).update(guild.name)
        self.query_one("#chat", RichLog).clear()

    def _add_tree_node(self, parent, node: TreeNode) -> None:
        label = channel_label(node.kind, node.name)
        if node.kind == "category":
            branch = parent.add(label, data=node, expand=True)
            if node.id is not None:
                self._tree_nodes[node.id] = node
            for child in node.children:
                self._add_tree_node(branch, child)
            return
        leaf = parent.add_leaf(label, data=node)
        if node.id is not None:
            self._tree_nodes[node.id] = node
        del leaf

    async def open_text(self, channel: ChannelInfo) -> None:
        if self.gateway is None:
            return
        self.current_channel = channel
        self.seen_ids.clear()
        title = channel_label("text", channel.name)
        self.query_one("#chat-title", Static).update(title)
        log = self.query_one("#chat", RichLog)
        log.clear()
        try:
            history = await self.gateway.history(channel.id)
        except Exception as exc:
            self.notify(str(exc), severity="error")
            return
        for message in history:
            self._write_message(message)
        self.query_one("#composer", Input).focus()

    async def join_channel(self, channel: ChannelInfo) -> None:
        if self.gateway is None or self.current_guild is None:
            return
        try:
            await self.gateway.join_voice(self.current_guild.id, channel.id, channel.name)
        except Exception as exc:
            self.notify(str(exc), severity="error")
            return
        self.voice.on_joined(self.current_guild.id, channel.id, channel.name)
        self._refresh_voicebar()
        self.notify(f"joined {channel.name}")

    def _write_message(self, message: ChatMessage) -> None:
        self.seen_ids.add(message.id)
        line = format_markup_line(message.author, message.content, message.timestamp)
        self.query_one("#chat", RichLog).write(line)

    def _refresh_voicebar(self) -> None:
        state = self.voice.state
        text = voice_bar_text(state.connected, state.muted, state.channel_name)
        self.query_one("#voicebar", Static).update(text)

    async def refresh_guilds(self) -> None:
        if self.gateway is None:
            return
        guilds = await self.gateway.guilds()
        listing = self.query_one("#guilds", ListView)
        listing.clear()
        for guild in guilds:
            await listing.append(GuildItem(guild))
        if guilds:
            listing.index = 0
            await self.open_guild(guilds[0])

    async def action_reload(self) -> None:
        if self.current_channel is not None and self.current_channel.kind == "text":
            await self.open_text(self.current_channel)
            self.notify("chat reloaded")
            return
        await self.refresh_guilds()
        self.notify("servers reloaded")

    async def action_mute(self) -> None:
        if self.gateway is None:
            return
        try:
            muted = self.voice.toggle_mute()
            await self.gateway.set_mute(muted)
        except Exception as exc:
            self.notify(str(exc), severity="error")
            return
        self._refresh_voicebar()

    async def action_join_voice(self) -> None:
        tree = self.query_one("#channels", Tree)
        cursor = tree.cursor_node
        if cursor is None or not isinstance(cursor.data, TreeNode):
            self.notify("select a voice channel", severity="warning")
            return
        node: TreeNode = cursor.data
        if node.kind != "voice" or node.channel is None:
            self.notify("select a voice channel", severity="warning")
            return
        await self.join_channel(node.channel)

    async def action_leave_voice(self) -> None:
        if self.gateway is None:
            return
        try:
            await self.gateway.leave_voice()
        except Exception as exc:
            self.notify(str(exc), severity="error")
            return
        self.voice.on_left()
        self._refresh_voicebar()
        self.notify("left call")

    def action_add_server(self) -> None:
        app_id = None
        if self.gateway is not None:
            app_id = self.gateway.application_id
        app_id = app_id or load_application_id()
        if not app_id:
            self.push_screen(
                InviteModal(
                    "Run once connected, then press a again.",
                    "Bot id is stored after the first successful login.",
                )
            )
            return
        self.push_screen(
            InviteModal(
                invite_url(app_id),
                "Open the URL, pick a server, then press r to reload.",
            )
        )

    def action_focus_nav(self) -> None:
        self.query_one("#channels", Tree).focus()


def run_app(*, demo: bool = False) -> None:
    PlaincordApp(demo=demo).run()
