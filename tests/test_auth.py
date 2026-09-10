from __future__ import annotations

from pathlib import Path

from plaincord.auth import (
    delete_token,
    load_application_id,
    load_token,
    looks_like_token,
    save_application_id,
    save_token,
)


def _isolate(tmp_path: Path, monkeypatch) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path))
    monkeypatch.delenv("PLAINCORD_TOKEN", raising=False)
    monkeypatch.setattr("plaincord.auth._keyring_get", lambda: None)
    monkeypatch.setattr("plaincord.auth._keyring_set", lambda token: False)
    monkeypatch.setattr("plaincord.auth._keyring_delete", lambda: None)


def test_looks_like_token() -> None:
    assert looks_like_token("x" * 50)
    assert not looks_like_token("short")
    assert not looks_like_token("x" * 50 + " has space")
    assert not looks_like_token("   ")


def test_save_and_load_token_file(tmp_path: Path, monkeypatch) -> None:
    _isolate(tmp_path, monkeypatch)
    token = "A" * 60
    save_token(token)
    path = tmp_path / "plaincord" / "token"
    assert path.is_file()
    assert oct(path.stat().st_mode)[-3:] == "600"
    assert load_token() == token


def test_env_token_wins(tmp_path: Path, monkeypatch) -> None:
    _isolate(tmp_path, monkeypatch)
    save_token("B" * 60)
    monkeypatch.setenv("PLAINCORD_TOKEN", "C" * 60)
    assert load_token() == "C" * 60


def test_delete_token(tmp_path: Path, monkeypatch) -> None:
    _isolate(tmp_path, monkeypatch)
    save_token("D" * 60)
    delete_token()
    assert load_token() is None


def test_reject_short_token(tmp_path: Path, monkeypatch) -> None:
    _isolate(tmp_path, monkeypatch)
    try:
        save_token("nope")
    except ValueError:
        return
    raise AssertionError("expected ValueError")


def test_application_id_roundtrip(tmp_path: Path, monkeypatch) -> None:
    _isolate(tmp_path, monkeypatch)
    assert load_application_id() is None
    save_application_id("42")
    assert load_application_id() == "42"
