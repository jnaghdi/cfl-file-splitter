#!/usr/bin/env python3
"""CFLSPLIT/1 inspector, verifier and joiner. Python 3.9+, standard library only.
No network use and no execution or automatic extraction of recovered content.
Save as cfl_join.py, or run this .txt file directly with python3.

  python3 CLAUDE_JOINER.txt inspect --parts /path/to/parts
  python3 CLAUDE_JOINER.txt verify  --parts /path/to/parts
  python3 CLAUDE_JOINER.txt join    --parts /path/to/parts --output /new/file.ext

The original attachment bytes, not an extracted text preview, are required.
SHA-256 provides integrity checks against the supplied hashes, not authorship.
"""
from __future__ import annotations

import argparse
import base64
import binascii
import hashlib
import json
import os
from pathlib import Path
import re
import sys
import tempfile
from typing import Any, BinaryIO, Iterator

MAGIC = b"CFLSPLIT/1\n"
HEADER_MAX = 8192
BLOCK = 1024 * 1024
MAX_PARTS = 1_000_000
HEX64 = re.compile(r"[0-9a-f]{64}\Z")
GLOBAL_FIELDS = (
    "format", "version", "set_id", "original_name", "original_size",
    "original_sha256", "encoding", "total_parts", "created_utc",
    "original_modified_utc",
)

class SplitError(Exception):
    """A malformed, incomplete, mixed or corrupt split set."""

def strict_json(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for k, v in pairs:
        if k in result:
            raise SplitError(f"Duplicate JSON key: {k}")
        result[k] = v
    return result

def integer(m: dict[str, Any], name: str, minimum: int = 0) -> int:
    v = m.get(name)
    if type(v) is not int or not minimum <= v <= 2**63 - 1:
        raise SplitError(f"Invalid integer field: {name}")
    return v

def read_part(path: Path) -> dict[str, Any] | None:
    if path.is_symlink() or not path.is_file():
        return None
    with path.open("rb") as f:
        if f.read(len(MAGIC)) != MAGIC:
            return None
        line = f.readline(HEADER_MAX)
        if not line.endswith(b"\n") or len(MAGIC) + len(line) + 1 > HEADER_MAX:
            raise SplitError(f"{path.name}: oversized/truncated header")
        if f.read(1) != b"\n":
            raise SplitError(f"{path.name}: missing blank header separator")
        try:
            m = json.loads(line, object_pairs_hook=strict_json)
        except (UnicodeDecodeError, ValueError) as e:
            raise SplitError(f"{path.name}: invalid header JSON: {e}") from e
        if not isinstance(m, dict) or m.get("format") != "CFLSPLIT" or type(m.get("version")) is not int or m["version"] != 1:
            raise SplitError(f"{path.name}: unsupported format/version")
        set_id = m.get("set_id", "")
        if not isinstance(set_id, str) or not re.fullmatch(r"[0-9a-fA-F]{32}", set_id):
            raise SplitError(f"{path.name}: invalid set ID")
        name = m.get("original_name", "")
        if (not isinstance(name, str) or not name or name in (".", "..") or
                name.strip() != name or name.endswith(".") or
                any(c in name for c in "/\\:\x00") or any(ord(c) < 32 for c in name)):
            raise SplitError(f"{path.name}: unsafe original filename")
        for field in ("original_sha256", "payload_sha256"):
            if not isinstance(m.get(field), str) or not HEX64.fullmatch(m[field]):
                raise SplitError(f"{path.name}: invalid {field}")
        size = integer(m, "original_size")
        offset = integer(m, "offset")
        length = integer(m, "payload_size")
        index = integer(m, "part_index", 1)
        count = integer(m, "total_parts", 1)
        if count > MAX_PARTS or index > count or offset > size or length > size - offset:
            raise SplitError(f"{path.name}: invalid index/count/byte range")
        if m.get("encoding") not in ("base64", "utf8", "binary"):
            raise SplitError(f"{path.name}: unsupported encoding")
        if not isinstance(m.get("created_utc"), str) or not isinstance(m.get("original_modified_utc"), str):
            raise SplitError(f"{path.name}: missing timestamps")
        start = f.tell()
        expected = ((length + 2) // 3 * 4) if m["encoding"] == "base64" else length
        if os.fstat(f.fileno()).st_size - start != expected:
            raise SplitError(f"{path.name}: transport length differs from header; truncated/altered/extra bytes")
    return {"metadata": m, "path": path, "start": start, "encoded_size": expected}

def inventory(folder: Path) -> list[dict[str, Any]]:
    if not folder.is_dir():
        raise SplitError("Parts directory does not exist")
    parts = []
    for path in sorted(folder.iterdir()):
        p = read_part(path)
        if p is not None:
            parts.append(p)
    if not parts:
        raise SplitError("No CFLSPLIT/1 parts found; original uploaded bytes are required")
    first = parts[0]["metadata"]
    seen: set[int] = set()
    for p in parts:
        m = p["metadata"]
        if m["set_id"] != first["set_id"]:
            raise SplitError("Mixed split sets: use a separate directory for each set")
        if any(m.get(k) != first.get(k) for k in GLOBAL_FIELDS):
            raise SplitError(f"{p['path'].name}: inconsistent metadata")
        index = m["part_index"]
        if index in seen:
            raise SplitError(f"Duplicate part {index}; do not silently discard duplicates")
        seen.add(index)
    if len(parts) != first["total_parts"]:
        missing = []
        for i in range(1, first["total_parts"] + 1):
            if i not in seen:
                missing.append(i)
                if len(missing) == 20:
                    break
        raise SplitError(f"Missing parts: have {len(parts)} of {first['total_parts']}; first missing indices: {missing}")
    parts.sort(key=lambda p: p["metadata"]["part_index"])
    offset = 0
    for p in parts:
        m = p["metadata"]
        if m["offset"] != offset:
            raise SplitError(f"Part {m['part_index']}: overlapping or missing byte range")
        if first["original_size"] > 0 and m["payload_size"] == 0:
            raise SplitError("Unexpected empty part")
        offset += m["payload_size"]
    if offset != first["original_size"]:
        raise SplitError("Part lengths do not total the original file size")
    return parts

def decoded_blocks(p: dict[str, Any]) -> Iterator[bytes]:
    current = read_part(p["path"])
    if current is None or current["metadata"] != p["metadata"]:
        raise SplitError("Part metadata changed after inventory")
    m = p["metadata"]
    with p["path"].open("rb") as f:
        f.seek(p["start"])
        remaining = p["encoded_size"]
        while remaining:
            b = f.read(min(BLOCK, remaining))  # BLOCK is a multiple of four.
            if not b:
                raise SplitError(f"Part {m['part_index']}: unexpected end of file")
            remaining -= len(b)
            if m["encoding"] == "base64":
                try:
                    data = base64.b64decode(b, validate=True)
                except binascii.Error as e:
                    raise SplitError(f"Part {m['part_index']}: invalid Base64: {e}") from e
                # Canonical encoding also rejects non-zero padding bits.
                if base64.b64encode(data) != b:
                    raise SplitError(f"Part {m['part_index']}: non-canonical Base64")
                yield data
            else:
                yield b
        if f.read(1):
            raise SplitError(f"Part {m['part_index']}: unexpected trailing data")

def consume(parts: list[dict[str, Any]], output: BinaryIO | None = None,
            quiet: bool = False) -> dict[str, Any]:
    whole = hashlib.sha256()
    total = 0
    for p in parts:
        ph = hashlib.sha256()
        count = 0
        for data in decoded_blocks(p):
            count += len(data)
            if count > p["metadata"]["payload_size"]:
                raise SplitError("Decoded data exceeds declared length")
            ph.update(data)
            whole.update(data)
            if output is not None:
                output.write(data)
        m = p["metadata"]
        if count != m["payload_size"] or ph.hexdigest() != m["payload_sha256"]:
            raise SplitError(f"Part {m['part_index']}: SHA-256/length mismatch")
        total += count
        if not quiet:
            print(f"Part {m['part_index']}/{m['total_parts']}: SHA-256 verified", file=sys.stderr)
    first = parts[0]["metadata"]
    if total != first["original_size"] or whole.hexdigest() != first["original_sha256"]:
        raise SplitError("Whole-file SHA-256/length mismatch")
    return {"status": "VERIFIED", "set_id": first["set_id"],
            "original_name": first["original_name"], "bytes": total,
            "parts": len(parts), "sha256": whole.hexdigest()}

def file_hash(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        while True:
            data = f.read(BLOCK)
            if not data:
                return h.hexdigest()
            h.update(data)

def join(parts: list[dict[str, Any]], destination: Path, quiet: bool = False) -> dict[str, Any]:
    if os.path.lexists(destination):
        raise SplitError("Destination already exists; choose a NEW path. Nothing was overwritten")
    if not destination.parent.is_dir():
        raise SplitError("Output parent directory does not exist")
    fd, tmp_name = tempfile.mkstemp(prefix=".cfl-rejoin-", suffix=".partial", dir=destination.parent)
    tmp = Path(tmp_name)
    try:
        with os.fdopen(fd, "wb") as out:
            result = consume(parts, out, quiet)
            out.flush()
            os.fsync(out.fileno())
        if file_hash(tmp) != result["sha256"]:
            raise SplitError("Reconstructed file failed read-back verification")
        try:
            os.link(tmp, destination)  # Atomic and refuses existing destination.
        except FileExistsError:
            raise SplitError("Destination appeared during joining; nothing was overwritten")
        except OSError:
            # Filesystem without hard links: create exclusively, copy, hash and sync.
            created = False
            try:
                with destination.open("xb") as out:
                    created = True
                    h = hashlib.sha256()
                    with tmp.open("rb") as source:
                        while True:
                            data = source.read(BLOCK)
                            if not data:
                                break
                            h.update(data)
                            out.write(data)
                    out.flush()
                    os.fsync(out.fileno())
                if h.hexdigest() != result["sha256"] or file_hash(destination) != result["sha256"]:
                    raise SplitError("Published output failed verification")
            except BaseException:
                if created:
                    destination.unlink(missing_ok=True)
                raise
        result["output"] = str(destination)
        return result
    finally:
        tmp.unlink(missing_ok=True)

def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("command", choices=["inspect", "verify", "join"])
    parser.add_argument("--parts", required=True, type=Path, help="One folder containing one complete split set")
    parser.add_argument("--output", type=Path, help="NEW output path; required for join")
    parser.add_argument("--quiet", action="store_true")
    args = parser.parse_args()
    if args.command == "join" and args.output is None:
        parser.error("join requires --output")
    try:
        parts = inventory(args.parts)
        if args.command == "inspect":
            result = {"status": "COMPLETE INVENTORY; PAYLOAD HASHES NOT YET VERIFIED",
                      "original": parts[0]["metadata"],
                      "files": [{"part_index": p["metadata"]["part_index"], "filename": p["path"].name,
                                 "offset": p["metadata"]["offset"], "payload_size": p["metadata"]["payload_size"]}
                                for p in parts]}
        elif args.command == "verify":
            result = consume(parts, quiet=args.quiet)
        else:
            result = join(parts, args.output.absolute(), quiet=args.quiet)
        print(json.dumps(result, indent=2, ensure_ascii=True))
        return 0
    except (SplitError, OSError, ValueError) as e:
        print(f"ERROR: {e}", file=sys.stderr)
        return 2
    except KeyboardInterrupt:
        print("Cancelled. No completed output was published.", file=sys.stderr)
        return 130

if __name__ == "__main__":
    raise SystemExit(main())
