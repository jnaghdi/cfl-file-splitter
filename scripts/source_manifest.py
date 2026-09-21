#!/usr/bin/env python3
"""Create/check the explicit, byte-exact source inventory. Python stdlib only.

Review new files before --write. This is integrity checking, not authorship,
malware scanning or proof that the files contain no confidential data.
"""
from __future__ import annotations
import argparse
import hashlib
from pathlib import Path, PurePosixPath
import re
import sys

ROOT = Path(__file__).resolve().parents[1]
MANIFEST = ROOT / 'SOURCE_SHA256SUMS.txt'
TOP_FILES = (
    'README.md', 'VERSION', 'go.mod', 'go.sum', 'LICENSE', 'CHANGELOG.md',
    'SECURITY.md', 'CONTRIBUTING.md', 'CLAUDE.md', 'THIRD_PARTY_NOTICES.md',
    '.gitignore', '.gitattributes', '.editorconfig',
)
SOURCE_DIRS = ('cmd', 'internal', 'tools', 'tests', 'scripts', 'examples',
               'docs', 'third-party', '.github')
SKIP_NAMES = {'__pycache__', '.DS_Store', 'Thumbs.db', '.git', '.venv'}


def digest(path: Path) -> str:
    h = hashlib.sha256()
    with path.open('rb') as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def safe_path(relative: str) -> Path:
    p = PurePosixPath(relative)
    if (not relative or p.is_absolute() or '\\' in relative or ':' in relative
            or '\n' in relative or '\r' in relative or '\x00' in relative
            or any(part in ('', '.', '..') for part in relative.split('/'))):
        raise ValueError(f'Unsafe source inventory path: {relative!r}')
    if relative not in TOP_FILES and p.parts[0] not in SOURCE_DIRS:
        raise ValueError(f'Path is outside source allowlist: {relative}')
    target = ROOT.joinpath(*p.parts)
    if not target.resolve().is_relative_to(ROOT.resolve()):
        raise ValueError(f'Path escapes repository: {relative}')
    current = target
    while current != ROOT:
        if current.is_symlink():
            raise ValueError(f'Symlinks are not accepted in source inventory: {relative}')
        current = current.parent
    return target


def source_files() -> list[Path]:
    found = [ROOT / name for name in TOP_FILES if (ROOT / name).is_file()]
    for directory in SOURCE_DIRS:
        base = ROOT / directory
        if not base.is_dir():
            continue
        for path in base.rglob('*'):
            rel = path.relative_to(ROOT)
            if any(part in SKIP_NAMES for part in rel.parts) or path.suffix in ('.pyc', '.pyo'):
                continue
            if path.is_symlink():
                raise ValueError(f'Symlink in source tree: {rel.as_posix()}')
            if path.is_file():
                safe_path(rel.as_posix())
                found.append(path)
    return sorted(found, key=lambda p: p.relative_to(ROOT).as_posix())


def entries() -> list[tuple[str, str]]:
    result = []
    seen = set()
    for line in MANIFEST.read_text(encoding='utf-8').splitlines():
        if not line or line.startswith('#'):
            continue
        m = re.fullmatch(r'([0-9a-f]{64})  (.+)', line)
        if not m:
            raise ValueError('Malformed SOURCE_SHA256SUMS.txt line')
        checksum, relative = m.groups()
        safe_path(relative)
        if relative.casefold() in seen:
            raise ValueError(f'Duplicate/case-colliding inventory entry: {relative}')
        seen.add(relative.casefold())
        result.append((checksum, relative))
    if not result:
        raise ValueError('Empty source inventory')
    return result


def verify() -> list[tuple[str, str]]:
    items = entries()
    recorded = {relative for _, relative in items}
    actual = {p.relative_to(ROOT).as_posix() for p in source_files()}
    if recorded != actual:
        raise ValueError(f'Source inventory mismatch. Unlisted: {sorted(actual-recorded)}; '
                         f'missing: {sorted(recorded-actual)}')
    for checksum, relative in items:
        path = safe_path(relative)
        if not path.is_file() or digest(path) != checksum:
            raise ValueError(f'Source hash mismatch: {relative}')
    return items


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    group = parser.add_mutually_exclusive_group()
    group.add_argument('--write', action='store_true', help='Recreate after reviewing source changes')
    group.add_argument('--check', action='store_true', help='Verify source inventory (default)')
    args = parser.parse_args()
    try:
        if args.write:
            files = source_files()
            lines = ['# CFL FileSplitter For Uploading Large Files To Claude source inventory; SHA-256, two spaces, relative path.',
                     '# The inventory does not authenticate its own origin. Review files before publishing.']
            lines += [f'{digest(p)}  {p.relative_to(ROOT).as_posix()}' for p in files]
            with MANIFEST.open('w', encoding='utf-8', newline='\n') as output:
                output.write('\n'.join(lines) + '\n')
            print(f'Wrote {len(files)} source checksums to {MANIFEST.name}')
        items = verify()
        print(f'PASS: {len(items)} source-file hashes match the complete source inventory.')
        return 0
    except (OSError, ValueError) as exc:
        print(f'ERROR: {exc}', file=sys.stderr)
        return 1

if __name__ == '__main__':
    raise SystemExit(main())
