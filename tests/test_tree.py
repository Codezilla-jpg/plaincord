from __future__ import annotations

from plaincord.models import ChannelInfo
from plaincord.tree import BOT_PERMISSIONS, build_channel_tree, invite_url


def test_invite_url_contains_bot_scope_and_permissions() -> None:
    url = invite_url("123456789")
    assert url.startswith("https://discord.com/oauth2/authorize?")
    assert "client_id=123456789" in url
    assert "scope=bot" in url
    assert f"permissions={BOT_PERMISSIONS}" in url


def test_channel_tree_groups_categories_and_sorts() -> None:
    channels = [
        ChannelInfo(2, "Voice", "category", None, 2),
        ChannelInfo(1, "Text", "category", None, 1),
        ChannelInfo(30, "lounge", "voice", 2, 0),
        ChannelInfo(20, "random", "text", 1, 1),
        ChannelInfo(10, "general", "text", 1, 0),
        ChannelInfo(5, "welcome", "text", None, 0),
        ChannelInfo(31, "afk", "voice", 2, 1),
    ]
    tree = build_channel_tree(channels)
    assert [node.name for node in tree] == ["welcome", "Text", "Voice"]
    assert tree[0].kind == "text"
    text_children = [child.name for child in tree[1].children]
    assert text_children == ["general", "random"]
    voice_children = [child.name for child in tree[2].children]
    assert voice_children == ["lounge", "afk"]
    assert all(child.kind == "voice" for child in tree[2].children)
