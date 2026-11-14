#!/usr/bin/env python3
"""Drive gh-sweep in a terminal and check the deletes it actually requests."""

import os
import pty
import select
import struct
import subprocess
import sys
import termios
import fcntl
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
ART = ROOT / "test" / "fixtures" / "artifacts.json"
CACHES = ROOT / "test" / "fixtures" / "caches.json"


def build_streams(tmp: Path) -> None:
    art = subprocess.check_output(
        ["bash", "-c", 'source "$1"; artifact_jq', "bash", str(ROOT / "gh-sweep")],
        text=True,
    )
    cache = subprocess.check_output(
        ["bash", "-c", 'source "$1"; cache_jq', "bash", str(ROOT / "gh-sweep")],
        text=True,
    )
    subprocess.check_call(["jq", "-r", art, str(ART)], stdout=open(tmp / "art.stream", "w"))
    subprocess.check_call(["jq", "-r", cache, str(CACHES)], stdout=open(tmp / "cache.stream", "w"))
    other = tmp / "other.stream"
    other.write_text(
        "T 1\n"
        "55\037notes\037100\0372026-11-01T00:00:00Z\0370\037main\0371\n",
        encoding="utf-8",
    )


def run_session(keys: bytes, tmp: Path, timeout: float = 8.0) -> tuple[int, str, str]:
    dump = tmp / "dump.txt"
    log = tmp / "gh.log"
    dump.write_text("", encoding="utf-8")
    log.write_text("", encoding="utf-8")
    bindir = tmp / "bin"
    bindir.mkdir(exist_ok=True)
    gh = bindir / "gh"
    if gh.exists() or gh.is_symlink():
        gh.unlink()
    gh.symlink_to(ROOT / "test" / "mock-gh")
    env = os.environ.copy()
    env["PATH"] = str(bindir) + os.pathsep + env.get("PATH", "")
    env["GH_MOCK_LOG"] = str(log)
    env["GH_MOCK_ART"] = str(tmp / "art.stream")
    env["GH_MOCK_CACHES"] = str(tmp / "cache.stream")
    env["GH_MOCK_ART_OTHER"] = str(tmp / "other.stream")
    env["GH_SWEEP_DUMP"] = str(dump)
    env["TERM"] = "xterm-256color"
    env["LANG"] = "C.UTF-8"
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 36, 110, 0, 0))
    proc = subprocess.Popen(
        [str(ROOT / "gh-sweep"), "--ascii", "--no-color", "acme/widgets", "other/lib"],
        stdin=slave,
        stdout=slave,
        stderr=slave,
        env=env,
        close_fds=True,
    )
    os.close(slave)
    sent = False
    output = bytearray()
    deadline = time.time() + timeout
    try:
        while time.time() < deadline:
            if proc.poll() is not None and sent:
                break
            ready, _, _ = select.select([master], [], [], 0.1)
            if ready:
                try:
                    chunk = os.read(master, 8192)
                except OSError:
                    break
                if not chunk:
                    if proc.poll() is not None:
                        break
                else:
                    output.extend(chunk)
            text = dump.read_text(encoding="utf-8", errors="replace")
            if not sent and "acme/widgets" in text and "projects" in text:
                os.write(master, keys)
                sent = True
            if proc.poll() is not None:
                break
        else:
            proc.kill()
    finally:
        if proc.poll() is None:
            proc.kill()
            proc.wait(timeout=2)
        os.close(master)
    code = proc.wait(timeout=2)
    return code, log.read_text(encoding="utf-8", errors="replace"), dump.read_text(
        encoding="utf-8", errors="replace"
    )


def deletes(log: str) -> list[str]:
    found = []
    for line in log.splitlines():
        if line.startswith("DELETE_ARTIFACT ") or line.startswith("DELETE_CACHE "):
            found.append(line)
    return found


def main() -> int:
    import tempfile

    failed = 0
    with tempfile.TemporaryDirectory(prefix="gh-sweep-pty-") as raw:
        tmp = Path(raw)
        build_streams(tmp)

        code, log, dump = run_session(b" dwidgets\rq", tmp)
        got = deletes(log)
        want = [
            "DELETE_CACHE acme/widgets 7",
            "DELETE_ARTIFACT acme/widgets 101",
            "DELETE_ARTIFACT acme/widgets 100",
            "DELETE_CACHE acme/widgets 8",
            "DELETE_CACHE acme/widgets 9",
            "DELETE_ARTIFACT acme/widgets 90",
            "DELETE_ARTIFACT acme/widgets 80",
        ]
        if code == 0 and got == want and "other/lib" not in "\n".join(got):
            print("ok  wipe one project, largest first")
        else:
            failed += 1
            print("FAIL wipe one project, largest first", file=sys.stderr)
            print("exit", code, file=sys.stderr)
            print(log, file=sys.stderr)
            print("--- dump tail ---", file=sys.stderr)
            print("\n".join(dump.splitlines()[-40:]), file=sys.stderr)

        code, log, dump = run_session(b"\r dwidgets\rq", tmp)
        got = deletes(log)
        want = [
            "DELETE_ARTIFACT acme/widgets 101",
            "DELETE_ARTIFACT acme/widgets 100",
        ]
        if code == 0 and got == want and "Type widgets" in dump:
            print("ok  pick one artifact group inside a project")
        else:
            failed += 1
            print("FAIL pick one artifact group inside a project", file=sys.stderr)
            print("exit", code, file=sys.stderr)
            print(log, file=sys.stderr)
            print("--- dump tail ---", file=sys.stderr)
            print("\n".join(dump.splitlines()[-50:]), file=sys.stderr)

        code, log, dump = run_session(b" dqq", tmp)
        got = deletes(log)
        if code == 0 and got == [] and "DELETE_" not in log:
            print("ok  cancel leaves the project alone")
        else:
            failed += 1
            print("FAIL cancel leaves the project alone", file=sys.stderr)
            print(log, file=sys.stderr)

    print(f"\n{failed} failed")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
