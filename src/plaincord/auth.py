from __future__ import annotations

import json
import os
from pathlib import Path

SERVICE = "plaincord"
USERNAME = "bot-token"
ENV_TOKEN = "PLAINCORD_TOKEN"


def config_dir() -> Path:
    xdg = os.environ.get("XDG_CONFIG_HOME")
    base = Path(xdg) if xdg else Path.home() / ".config"
    return base / "plaincord"


def token_path() -> Path:
    return config_dir() / "token"


def config_path() -> Path:
    return config_dir() / "config.json"


def looks_like_token(token: str) -> bool:
    value = token.strip()
    return len(value) >= 50 and " " not in value and "\n" not in value


def _keyring_get() -> str | None:
    try:
        import keyring

        return keyring.get_password(SERVICE, USERNAME)
    except Exception:
        return None


def _keyring_set(token: str) -> bool:
    try:
        import keyring

        keyring.set_password(SERVICE, USERNAME, token)
        return True
    except Exception:
        return False


def _keyring_delete() -> None:
    try:
        import keyring

        keyring.delete_password(SERVICE, USERNAME)
    except Exception:
        return


def load_token() -> str | None:
    env = os.environ.get(ENV_TOKEN)
    if env and env.strip():
        return env.strip()
    stored = _keyring_get()
    if stored:
        return stored.strip()
    path = token_path()
    if path.is_file():
        value = path.read_text(encoding="utf-8").strip()
        return value or None
    return None


def save_token(token: str) -> None:
    value = token.strip()
    if not looks_like_token(value):
        raise ValueError("Token looks invalid")
    if _keyring_set(value):
        return
    path = token_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(value, encoding="utf-8")
    path.chmod(0o600)


def delete_token() -> None:
    _keyring_delete()
    path = token_path()
    if path.is_file():
        path.unlink()


def load_config() -> dict:
    path = config_path()
    if not path.is_file():
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return {}
    return data if isinstance(data, dict) else {}


def save_config(data: dict) -> None:
    path = config_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")
    path.chmod(0o600)


def save_application_id(application_id: str) -> None:
    data = load_config()
    data["application_id"] = str(application_id)
    save_config(data)


def load_application_id() -> str | None:
    value = load_config().get("application_id")
    return str(value) if value else None
