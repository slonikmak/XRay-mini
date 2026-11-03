#!/usr/bin/env python3
"""Require complete native loopback execution in a Go test2json report."""
from __future__ import annotations

import argparse
import json
from pathlib import Path
import sys
from typing import Iterable

PACKAGE = "github.com/slonikmak/XRay-mini/internal/core"
ROOT = "TestNativeWindowsLoopback"
TESTS = (ROOT, *(f"{ROOT}/{name}" for name in ("raw-reality", "xhttp-reality", "xhttp-tls")))
FAILURE = "[NATIVE_LOOPBACK_FAILED] Expected package completion and run/pass for the root and all three transports; skips, failures and incomplete reports are rejected."


def verify(lines: Iterable[str]) -> None:
    states = dict.fromkeys(TESTS)
    started = completed = False
    for line in lines:
        event = json.loads(line)
        if not isinstance(event, dict) or event.get("Package") != PACKAGE or completed:
            raise ValueError(FAILURE)
        action, test = event.get("Action"), event.get("Test")
        if action == "start" and test is None and not started:
            started = True
        elif not started:
            raise ValueError(FAILURE)
        elif action == "output":
            if not isinstance(event.get("Output"), str) or (test is not None and
                    (not isinstance(test, str) or test not in states or states[test] is None)):
                raise ValueError(FAILURE)
        elif action == "run" and isinstance(test, str) and test in states and states[test] is None:
            if test != ROOT and states[ROOT] != "run":
                raise ValueError(FAILURE)
            states[test] = "run"
        elif action == "pass" and isinstance(test, str) and test in states and states[test] == "run":
            if test == ROOT and any(states[name] != "pass" for name in TESTS[1:]):
                raise ValueError(FAILURE)
            states[test] = "pass"
        elif action == "pass" and test is None and all(value == "pass" for value in states.values()):
            completed = True
        else:
            raise ValueError(FAILURE)
    if not completed:
        raise ValueError(FAILURE)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("report", type=Path)
    args = parser.parse_args(argv)
    try:
        with args.report.open(encoding="utf-8-sig") as report:
            verify(report)
    except (OSError, UnicodeError, ValueError):
        print(FAILURE, file=sys.stderr)
        return 1
    print("Native loopback report: root, raw-reality, xhttp-reality, xhttp-tls and package passed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
