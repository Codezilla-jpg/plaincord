from __future__ import annotations

import pytest

from plaincord.cli import main


def test_help_exits_zero() -> None:
    with pytest.raises(SystemExit) as exc:
        main(["--help"])
    assert exc.value.code == 0


def test_version_exits_zero() -> None:
    with pytest.raises(SystemExit) as exc:
        main(["--version"])
    assert exc.value.code == 0


def test_invite_without_id(tmp_path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path))
    monkeypatch.setattr("plaincord.cli.load_application_id", lambda: None)
    assert main(["invite"]) == 1
    err = capsys.readouterr().err
    assert "No bot id" in err


def test_invite_prints_url(monkeypatch, capsys) -> None:
    monkeypatch.setattr("plaincord.cli.load_application_id", lambda: "99")
    assert main(["invite"]) == 0
    out = capsys.readouterr().out
    assert "client_id=99" in out
    assert "scope=bot" in out


def test_logout(tmp_path, monkeypatch, capsys) -> None:
    monkeypatch.setenv("XDG_CONFIG_HOME", str(tmp_path))
    called = {"n": 0}

    def _delete() -> None:
        called["n"] += 1

    monkeypatch.setattr("plaincord.cli.delete_token", _delete)
    assert main(["logout"]) == 0
    assert called["n"] == 1
    assert "removed" in capsys.readouterr().out.lower()
