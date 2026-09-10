from __future__ import annotations

from plaincord.models import ChannelInfo, TreeNode

# View Channel, Send Messages, Read Message History, Connect, Speak, Use VAD
BOT_PERMISSIONS = (
    (1 << 10) | (1 << 11) | (1 << 16) | (1 << 20) | (1 << 21) | (1 << 25)
)


def invite_url(client_id: int | str) -> str:
    return (
        "https://discord.com/oauth2/authorize"
        f"?client_id={client_id}"
        f"&permissions={BOT_PERMISSIONS}"
        "&integration_type=0"
        "&scope=bot"
    )


def build_channel_tree(channels: list[ChannelInfo]) -> tuple[TreeNode, ...]:
    categories = sorted(
        (c for c in channels if c.kind == "category"),
        key=lambda c: (c.position, c.id),
    )
    by_category: dict[int | None, list[ChannelInfo]] = {}
    for channel in channels:
        if channel.kind == "category":
            continue
        by_category.setdefault(channel.category_id, []).append(channel)
    for group in by_category.values():
        group.sort(key=lambda c: (0 if c.kind == "text" else 1, c.position, c.id))

    nodes: list[TreeNode] = []
    for channel in by_category.get(None, []):
        nodes.append(_leaf(channel))
    for category in categories:
        children = tuple(_leaf(c) for c in by_category.get(category.id, []))
        nodes.append(
            TreeNode(
                id=category.id,
                name=category.name,
                kind="category",
                children=children,
                channel=category,
            )
        )
    return tuple(nodes)


def _leaf(channel: ChannelInfo) -> TreeNode:
    return TreeNode(
        id=channel.id,
        name=channel.name,
        kind=channel.kind,
        children=(),
        channel=channel,
    )
