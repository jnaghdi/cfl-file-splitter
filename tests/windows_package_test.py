#!/usr/bin/env python3
"""Inspect a built Windows ZIP without executing the Windows applications."""
from __future__ import annotations
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath
import struct
import sys
import zipfile

NAME = 'CFL FileSplitter For Uploading Large Files To Claude'
ROOT = Path(__file__).resolve().parents[1]


def check_package(archive: Path) -> list[str]:
    passed: list[str] = []
    def require(condition: bool, description: str) -> None:
        if not condition:
            raise ValueError(description)
        passed.append('PASS: '+description)
    sidecar=archive.with_suffix(archive.suffix+'.sha256')
    expected=sidecar.read_text().split()[0]
    require(hashlib.sha256(archive.read_bytes()).hexdigest()==expected, 'external ZIP SHA-256')
    with zipfile.ZipFile(archive) as z:
        names=z.namelist()
        require(len(set(names))==len(names), 'unique ZIP members')
        require(all(not PurePosixPath(n).is_absolute() and '..' not in PurePosixPath(n).parts and '\\' not in n for n in names), 'safe relative ZIP paths')
        require(sum(i.file_size for i in z.infolist())<128*1024*1024,'package within expected inspection size')
        require(z.testzip() is None, 'ZIP CRCs')
        version=(ROOT/'VERSION').read_text().strip()
        prefix='CFL_FileSplitter_v'+version+'/'
        require(all(n.startswith(prefix) for n in names), 'single expected versioned root folder')
        data={n[len(prefix):]:z.read(n) for n in names if not n.endswith('/')}
        required=(NAME+'.exe','CFL_FileSplitter_CLI.exe','README.md','LICENSE','START_HERE.txt','BUILD_INFO.json','SHA256SUMS.txt','SOURCE_SHA256SUMS.txt','THIRD_PARTY_NOTICES.md','runtime-notices/GO-LICENSE.txt','CLAUDE_JOINER.txt')
        require(all(p in data for p in required), 'all required release files')
        require('CFL_File_Splitter.exe' not in data,'obsolete root GUI filename absent')
        readme=data['README.md'].decode('utf-8')
        require(readme.splitlines()[0]=='# '+NAME and 'https://cflab.uk' in readme and 'https://e-discovery.uk' in readme,'README application name and both websites')
        require(data['LICENSE']==(ROOT/'LICENSE').read_bytes()==data['internal/core/assets/LICENSE.txt'],'MIT licence equals source and embedded copy')
        require(data['CLAUDE_JOINER.txt']==data['tools/claude_joiner.py']==data['internal/core/assets/CLAUDE_JOINER.txt'],'all bundled Python helper copies identical')
        expected_members=set()
        for line in data['SHA256SUMS.txt'].decode().splitlines():
            sha,relative=line.split('  ',1)
            if hashlib.sha256(data[relative]).hexdigest()!=sha:
                raise ValueError('Package hash mismatch: '+relative)
            expected_members.add(relative)
        require(expected_members==set(data)-{'SHA256SUMS.txt'},'complete internal SHA-256 inventory')
        for line in data['SOURCE_SHA256SUMS.txt'].decode().splitlines():
            if not line or line.startswith('#'):continue
            sha,relative=line.split('  ',1)
            if hashlib.sha256(data[relative]).hexdigest()!=sha:
                raise ValueError('Source inventory mismatch: '+relative)
        passed.append('PASS: every embedded source checksum')
        info=json.loads(data['BUILD_INFO.json'])
        require(info['project']==NAME and info['version']==version and info['license']=='MIT','build metadata name/version/licence')
        require(info['websites']==['https://cflab.uk','https://e-discovery.uk'],'build metadata websites')
        for exe,subsystem in ((NAME+'.exe',2),('CFL_FileSplitter_CLI.exe',3)):
            b=data[exe]
            offset=struct.unpack_from('<I',b,0x3c)[0]
            require(b[:2]==b'MZ' and b[offset:offset+6]==b'PE\x00\x00\x64\x86','Windows x64 PE: '+exe)
            optional=offset+24
            require(struct.unpack_from('<H',b,optional)[0]==0x20b and struct.unpack_from('<H',b,optional+68)[0]==subsystem,'correct GUI/console subsystem: '+exe)
            require(NAME.encode() in b and b'https://cflab.uk' in b and b'https://e-discovery.uk' in b,'compiled branding strings: '+exe)
            require(info['binary_sha256'][exe]==hashlib.sha256(b).hexdigest(),'binary hash matches build metadata: '+exe)
        require(not info['interactive_windows_gui_tested_by_packager'] and not info['claude_upload_tested_by_packager'] and not info['authenticode_signed_by_packager'],'no unsupported signing/interactive-test claims')
    return passed


def main() -> int:
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('archive',type=Path)
    args=parser.parse_args()
    try:
        passed=check_package(args.archive.resolve())
    except (OSError,ValueError,KeyError,struct.error,zipfile.BadZipFile) as exc:
        print('FAIL: '+str(exc),file=sys.stderr);return 1
    print('\n'.join(passed))
    print(f'PASS: {len(passed)} Windows package static checks. No Windows code executed.')
    return 0

if __name__=='__main__':raise SystemExit(main())
