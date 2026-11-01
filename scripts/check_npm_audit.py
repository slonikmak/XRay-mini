#!/usr/bin/env python3
"""Retain full/prod npm reports; fail closed on high/critical findings or failed audits."""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys

ROOT = Path(__file__).resolve().parent.parent


def classify(report: object, returncode: int) -> tuple[int, str]:
    if not isinstance(report, dict) or "error" in report or returncode not in (0, 1):
        return 2, "Audit unavailable or failed; no clean result established"
    try:
        counts = report["metadata"]["vulnerabilities"]
        values = [counts[key] for key in ("info", "low", "moderate", "high", "critical", "total")]
        if any(type(n) is not int or n < 0 for n in values):
            raise ValueError
        if sum(values[:5]) != values[5]:
            raise ValueError
    except (KeyError, TypeError, ValueError):
        return 2, "Incomplete or invalid vulnerability counts"
    blocked = counts["high"] + counts["critical"] > 0
    # With audit-level=high, a nonzero result without high/critical is unexplained.
    if returncode != int(blocked):
        return 2, "Exit code and reported findings disagree"
    summary = ", ".join(f"{key}={counts[key]}" for key in ("low", "moderate", "high", "critical"))
    return (1 if blocked else 0), summary


def audit(frontend: Path, output: Path) -> int:
    output.mkdir(parents=True, exist_ok=True)
    summary = ["# Dependency audit", "", "Release gate: no high/critical findings in either scope; registry errors fail closed.", ""]
    npm = shutil.which("npm.cmd" if sys.platform == "win32" else "npm")
    lock = frontend / "package-lock.json"
    if not npm or not lock.is_file():
        summary.append("UNAVAILABLE: npm or package-lock.json is missing.")
        (output / "SUMMARY.md").write_text("\n".join(summary) + "\n", encoding="utf-8")
        return 2
    original_lock = lock.read_bytes()
    metadata = {"lockfile_sha256": hashlib.sha256(original_lock).hexdigest(), "runs": {}}
    outcomes = []
    for scope, flags in (("all", ["--include=dev", "--include=optional", "--include=peer"]), ("production", ["--omit=dev", "--include=optional", "--include=peer"])):
        command = [npm, "audit", "--json", "--package-lock-only", "--ignore-scripts", "--audit-level=high", *flags]
        try:
            run = subprocess.run(command, cwd=frontend, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=120)
            (output / f"{scope}.json").write_text(run.stdout, encoding="utf-8")
            (output / f"{scope}.stderr.txt").write_text(run.stderr, encoding="utf-8")
            try:
                report = json.loads(run.stdout)
            except json.JSONDecodeError:
                report = None
            result, detail = classify(report, run.returncode)
            metadata["runs"][scope] = {"arguments": command[1:], "exit_code": run.returncode, "gate_code": result}
        except (OSError, subprocess.TimeoutExpired) as error:
            result, detail = 2, f"Audit did not complete ({type(error).__name__})"
            metadata["runs"][scope] = {"arguments": command[1:], "gate_code": result, "error": type(error).__name__}
        outcomes.append(result)
        summary.append(f"- **{scope}: {'PASS' if result == 0 else 'FINDINGS' if result == 1 else 'UNAVAILABLE'}** — {detail}")
    if lock.read_bytes() != original_lock:
        outcomes.append(2)
        summary.append("- UNAVAILABLE: lockfile changed during audit; inspect the working tree.")
    (output / "metadata.json").write_text(json.dumps(metadata, indent=2) + "\n", encoding="utf-8")
    (output / "SUMMARY.md").write_text("\n".join(summary) + "\n", encoding="utf-8")
    print("\n".join(summary))
    return max(outcomes)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--frontend", default=ROOT / "frontend", type=Path)
    parser.add_argument("--output", default=ROOT / "artifacts/dependency-audit", type=Path)
    args = parser.parse_args()
    try:
        return audit(args.frontend, args.output)
    except OSError as error:
        print(f"Audit unavailable ({type(error).__name__}); no clean result established", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
