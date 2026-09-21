#!/usr/bin/env python3
"""Package reviewed source and previously built Windows x64 binaries locally."""
from __future__ import annotations
import argparse
from datetime import datetime, timezone
import json
import os
from pathlib import Path
import re
import shutil
import struct
import subprocess
import sys
import zipfile
from source_manifest import ROOT, MANIFEST, digest, safe_path, verify


def command(*args: str) -> str:
    return subprocess.check_output(args, cwd=ROOT, text=True, encoding='utf-8',
                                   stderr=subprocess.STDOUT).strip()


def validate_pe(path: Path) -> None:
    with path.open('rb') as f:
        if f.read(2) != b'MZ':
            raise ValueError(f'Not a Windows executable: {path.name}')
        f.seek(0x3c)
        offset_data = f.read(4)
        if len(offset_data) != 4:
            raise ValueError(f'Truncated executable: {path.name}')
        f.seek(struct.unpack('<I', offset_data)[0])
        signature = f.read(6)
        if signature != b'PE\x00\x00\x64\x86':
            raise ValueError(f'Expected Windows x64 PE: {path.name}')


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output-dir', type=Path, default=ROOT/'dist')
    args = parser.parse_args()
    created = False
    sidecar_created = False
    archive = None
    sidecar = None
    try:
        items = verify()
        version = (ROOT/'VERSION').read_text().strip()
        if not re.fullmatch(r'[0-9]+\.[0-9]+\.[0-9]+', version):
            raise ValueError('Invalid VERSION')
        binaries = [ROOT/'bin'/name for name in ('CFL_File_Splitter.exe', 'CFL_Splitter_CLI.exe')]
        for binary in binaries:
            validate_pe(binary)
        if not shutil.which('go'):
            raise ValueError('Go is required to record binary build metadata and its licence.')
        build_info = {
            'project': 'CFL File Splitter', 'version': version,
            'packaged_utc': datetime.now(timezone.utc).isoformat(),
            'packaging_go_version': command('go', 'version'),
            'binary_build_metadata': {p.name: command('go', 'version', '-m', str(p)) for p in binaries},
            'binary_sha256': {p.name: digest(p) for p in binaries},
            'source_inventory_sha256': digest(MANIFEST),
            'git_commit': None, 'git_worktree_dirty': None,
            'authenticode_signed_by_packager': False,
            'interactive_windows_gui_tested_by_packager': False,
            'claude_upload_tested_by_packager': False,
        }
        if shutil.which('git') and (ROOT/'.git').exists():
            try:
                build_info['git_commit'] = command('git', 'rev-parse', 'HEAD')
                build_info['git_worktree_dirty'] = bool(command('git', 'status', '--porcelain'))
            except subprocess.CalledProcessError:
                pass
        go_license = Path(command('go', 'env', 'GOROOT'))/'LICENSE'
        if not go_license.is_file():
            raise ValueError('Cannot locate the packaging Go toolchain licence.')
        prefix = f'CFL_File_Splitter_v{version}/'
        start = f"""CFL FILE SPLITTER {version} - WINDOWS X64\n\nExtract the ZIP before running CFL_File_Splitter.exe.\nNo Python/.NET/Go runtime is required to run the EXEs.\n\nChoose a source file, an existing output parent folder and maximum piece size.\nLeave Output format at Auto (recommended). Start with fictional data in examples/.\nClick Split & verify. Auto uses readable UTF-8 when suitable, otherwise lossless Base64.\nNUL bytes and UTF-16/binary content are preserved, not removed or converted.\nRead docs/NUL_BYTE_FIX.md for the 1.0.1 fix and the included UTF-16 regression example.\nREADME.md and docs/ contain usage, integrity, build and privacy information.\nCLAUDE_JOINER.txt is the Python helper; the app also emits a copy in each split set.\n\nUnsigned build. Read docs/VALIDATION.md for tests actually performed.\nCompiling a GUI is not testing it interactively. No Claude upload is guaranteed.\nSHA256SUMS.txt checks package contents; it is not a digital signature.\nBUILD_INFO.json records the available build metadata.\nNever upload sensitive files without appropriate authorisation.\n"""
        content = {relative: safe_path(relative).read_bytes() for _, relative in items}
        content[MANIFEST.name] = MANIFEST.read_bytes()
        for binary in binaries:
            content[binary.name] = binary.read_bytes()
        content['CLAUDE_JOINER.txt'] = (ROOT/'tools/claude_joiner.py').read_bytes()
        content['START_HERE.txt'] = start.encode('utf-8')
        content['BUILD_INFO.json'] = (json.dumps(build_info, indent=2)+'\n').encode('utf-8')
        content['runtime-notices/GO-LICENSE.txt'] = go_license.read_bytes()
        import hashlib
        content['SHA256SUMS.txt'] = ('\n'.join(
            f'{hashlib.sha256(data).hexdigest()}  {relative}'
            for relative, data in sorted(content.items()))+'\n').encode('utf-8')
        out_dir = args.output_dir.resolve()
        out_dir.mkdir(parents=True, exist_ok=True)
        archive = out_dir/f'CFL_File_Splitter_Windows_v{version}.zip'
        sidecar = archive.with_suffix(archive.suffix+'.sha256')
        if archive.exists() or sidecar.exists():
            raise FileExistsError('Release ZIP/checksum exists; use a new output directory. No overwrite.')
        with archive.open('xb') as stream:
            created = True
            with zipfile.ZipFile(stream, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as z:
                for relative, data in sorted(content.items()):
                    z.writestr(prefix+relative, data)
        with sidecar.open('x', encoding='utf-8', newline='\n') as stream:
            sidecar_created = True
            stream.write(f'{digest(archive)}  {archive.name}\n')
        print(f'Created {archive}')
        print(f'Created {sidecar}')
        return 0
    except (OSError, ValueError, subprocess.CalledProcessError) as exc:
        if created and archive is not None:
            archive.unlink(missing_ok=True)
        if sidecar_created and sidecar is not None:
            sidecar.unlink(missing_ok=True)
        print(f'ERROR: {exc}', file=sys.stderr)
        return 1

if __name__ == '__main__':
    raise SystemExit(main())
