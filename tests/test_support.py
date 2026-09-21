"""Shared helpers for standalone synthetic integration tests (standard library)."""
from __future__ import annotations
import atexit
import os
from pathlib import Path
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
HELPER = ROOT / "tools" / "claude_joiner.py"
_TEMP = None
_CLI = None

def native_cli() -> str:
    global _TEMP, _CLI
    if _CLI:
        return _CLI
    configured = os.environ.get("CFL_CLI")
    if configured:
        candidate = Path(configured).expanduser().resolve()
        if not candidate.is_file():
            raise RuntimeError(f"CFL_CLI is not a file: {candidate}")
        _CLI = str(candidate)
        return _CLI
    if not shutil.which("go"):
        raise RuntimeError("Install Go, or set CFL_CLI to an existing host-native CLI.")
    _TEMP = tempfile.TemporaryDirectory(prefix="cfl-native-test-")
    atexit.register(_TEMP.cleanup)
    target = Path(_TEMP.name) / ("cflsplit.exe" if os.name == "nt" else "cflsplit")
    env = os.environ.copy()
    # Do not accidentally cross-compile the executable that the test will run.
    env.pop("GOOS", None)
    env.pop("GOARCH", None)
    env["CGO_ENABLED"] = "0"
    subprocess.run(["go", "build", "-trimpath", "-buildvcs=false", "-o",
                    str(target), "./cmd/cflcli"], cwd=ROOT, env=env, check=True)
    _CLI = str(target)
    return _CLI
