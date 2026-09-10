from __future__ import annotations

import argparse
import getpass
import sys

from plaincord import __version__
from plaincord.auth import delete_token, load_application_id, save_token
from plaincord.tree import invite_url


def cmd_login() -> int:
    print("Create a bot: https://discord.com/developers/applications")
    print("Enable Privileged Gateway Intent: Message Content.")
    print("Then paste the bot token (input is hidden).")
    token = getpass.getpass("Bot token: ")
    try:
        save_token(token)
    except ValueError as exc:
        print(str(exc), file=sys.stderr)
        return 1
    print("Saved. Start with: plaincord")
    return 0


def cmd_logout() -> int:
    delete_token()
    print("Token removed.")
    return 0


def cmd_invite() -> int:
    app_id = load_application_id()
    if not app_id:
        print("No bot id stored yet. Run plaincord once, then retry.", file=sys.stderr)
        return 1
    print(invite_url(app_id))
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        prog="plaincord",
        description="Clean Discord TUI — servers, channels, chat, and voice.",
    )
    parser.add_argument("--demo", action="store_true", help="Run with sample data (no token)")
    parser.add_argument("--version", action="version", version=f"plaincord {__version__}")
    sub = parser.add_subparsers(dest="cmd")
    sub.add_parser("login", help="Store a Discord bot token")
    sub.add_parser("logout", help="Remove the stored token")
    sub.add_parser("invite", help="Print the bot invite URL")
    args = parser.parse_args(argv)

    if args.cmd == "login":
        return cmd_login()
    if args.cmd == "logout":
        return cmd_logout()
    if args.cmd == "invite":
        return cmd_invite()

    from plaincord.app import run_app

    run_app(demo=bool(args.demo))
    return 0
