#!/usr/bin/env python3
"""Cursor hook: refuse direct or forced updates to main/master.

beforeShellExecution and preToolUse (Shell) both call this. gh pr merge is
allowed; git push onto main/master is not.
"""

from __future__ import annotations

import json
import os
import subprocess
import sys

PROTECTED = frozenset({"main", "master"})
GIT_GLOBALS_WITH_VALUE = frozenset(
    {"-c", "-C", "--git-dir", "--work-tree", "--namespace"}
)
DENY_REASON = (
    "Direct update of main/master is refused. "
    "Open a PR with gh pr create, then merge with gh pr merge."
)


def extract_command(payload: dict) -> str:
    command = payload.get("command")
    if isinstance(command, str) and command.strip():
        return command
    tool_input = payload.get("tool_input")
    if isinstance(tool_input, dict):
        inner = tool_input.get("command")
        if isinstance(inner, str):
            return inner
    if isinstance(tool_input, str) and tool_input.strip().startswith("{"):
        try:
            parsed = json.loads(tool_input)
        except json.JSONDecodeError:
            parsed = None
        if isinstance(parsed, dict) and isinstance(parsed.get("command"), str):
            return parsed["command"]
    arguments = payload.get("arguments")
    if isinstance(arguments, dict) and isinstance(arguments.get("command"), str):
        return arguments["command"]
    return ""


def _split_invocations(command: str) -> list[str]:
    chunks: list[str] = []
    buf: list[str] = []
    in_single = False
    in_double = False
    i = 0
    while i < len(command):
        ch = command[i]
        nxt = command[i + 1] if i + 1 < len(command) else ""
        if ch == "'" and not in_double:
            in_single = not in_single
            buf.append(ch)
            i += 1
            continue
        if ch == '"' and not in_single:
            in_double = not in_double
            buf.append(ch)
            i += 1
            continue
        if not in_single and not in_double:
            if ch in {";", "\n"} or (ch == "&" and nxt != "&") or (
                ch == "|" and nxt != "|"
            ):
                chunk = "".join(buf).strip()
                if chunk:
                    chunks.append(chunk)
                buf = []
                i += 1
                continue
            if (ch == "&" and nxt == "&") or (ch == "|" and nxt == "|"):
                chunk = "".join(buf).strip()
                if chunk:
                    chunks.append(chunk)
                buf = []
                i += 2
                continue
        buf.append(ch)
        i += 1
    chunk = "".join(buf).strip()
    if chunk:
        chunks.append(chunk)
    return chunks


def _tokens(fragment: str) -> list[str]:
    try:
        import shlex

        return shlex.split(fragment, posix=True)
    except ValueError:
        return fragment.split()


def _exe_name(token: str) -> str:
    name = token.replace("\\", "/").rsplit("/", 1)[-1].lower()
    if name.endswith(".exe"):
        name = name[:-4]
    return name


def _git_push_args(tokens: list[str]) -> list[str] | None:
    i = 0
    if tokens and tokens[0].lower() == "sudo":
        i = 1
    if i >= len(tokens) or _exe_name(tokens[i]) != "git":
        return None
    i += 1
    while i < len(tokens):
        token = tokens[i]
        lower = token.lower()
        if lower == "push":
            return tokens[i + 1 :]
        if lower in GIT_GLOBALS_WITH_VALUE:
            i += 2
            continue
        if token.startswith("-"):
            i += 1
            continue
        return None
    return None


def _dest_names(spec: str) -> set[str]:
    names: set[str] = set()
    body = spec.lstrip("+")
    dst = body.split(":", 1)[1] if ":" in body else body
    dst = dst.strip()
    if not dst:
        return names
    names.add(dst)
    names.add(dst.rsplit("/", 1)[-1])
    if dst.startswith("refs/heads/"):
        names.add(dst[len("refs/heads/") :])
    return names


def _explicit_refspecs(push_args: list[str]) -> tuple[bool, list[str]]:
    """Return (mirror_or_all, refspecs). Empty refspecs means implicit HEAD/upstream."""
    flags_done = False
    remote: str | None = None
    specs: list[str] = []
    mirror_or_all = False
    i = 0
    while i < len(push_args):
        token = push_args[i]
        lower = token.lower()
        if not flags_done and token.startswith("-"):
            if lower in {"--all", "--mirror"}:
                mirror_or_all = True
            if lower in {"--repo", "--exec", "--signed"}:
                i += 2
                continue
            i += 1
            continue
        flags_done = True
        if remote is None:
            remote = token
            i += 1
            continue
        specs.append(token)
        i += 1
    if remote is not None and not specs:
        if remote.lower() in PROTECTED or remote.upper() == "HEAD" or ":" in remote:
            specs = [remote]
    return mirror_or_all, specs


def decide(command: str, current_branch: str) -> str | None:
    """Return a deny reason, or None to allow."""
    if not command.strip():
        return None
    for fragment in _split_invocations(command):
        tokens = _tokens(fragment)
        push_args = _git_push_args(tokens)
        if push_args is None:
            continue
        mirror_or_all, specs = _explicit_refspecs(push_args)
        if mirror_or_all:
            return DENY_REASON
        dests: set[str] = set()
        if not specs:
            dests.add(current_branch.lower())
        for spec in specs:
            names = _dest_names(spec)
            upper = {n.upper() for n in names}
            if "HEAD" in upper or spec.upper() in {"HEAD", "+HEAD"}:
                dests.add(current_branch.lower())
            dests.update(n.lower() for n in names if n.upper() != "HEAD")
        if dests & PROTECTED:
            return DENY_REASON
    return None


def current_branch(cwd: str) -> str:
    try:
        completed = subprocess.run(
            ["git", "rev-parse", "--abbrev-ref", "HEAD"],
            cwd=cwd or None,
            capture_output=True,
            text=True,
            timeout=5,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        return ""
    return completed.stdout.strip()


def main() -> int:
    try:
        raw = sys.stdin.read()
        payload = json.loads(raw) if raw.strip() else {}
    except json.JSONDecodeError:
        payload = {}
    if not isinstance(payload, dict):
        payload = {}
    command = extract_command(payload)
    cwd = payload.get("cwd") if isinstance(payload.get("cwd"), str) else os.getcwd()
    branch = current_branch(cwd)
    reason = decide(command, branch)
    if reason is None:
        sys.stdout.write(json.dumps({"permission": "allow"}))
        return 0
    sys.stdout.write(
        json.dumps(
            {
                "permission": "deny",
                "user_message": reason,
                "agent_message": reason,
            }
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
