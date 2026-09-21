"""Regression tests for v1.0.1 Auto mode using the actual CLI and Python joiner."""
from __future__ import annotations
import hashlib
import importlib.util
import json
from pathlib import Path
import random
import subprocess
import tempfile
from test_support import HELPER, native_cli

spec = importlib.util.spec_from_file_location('auto_regression_joiner', HELPER)
joiner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(joiner)
cli = native_cli()
checks: list[str] = []
text = 'Fictional regression data. English, سلام, 中文, 😀\r\n'
cases = [
    ('empty', b'', 'utf8'),
    ('ascii', b'Harmless sample.\r\n'*30, 'utf8'),
    ('utf8-unicode', (text*30).encode('utf-8'), 'utf8'),
    ('utf8-bom', b'\xef\xbb\xbf'+text.encode('utf-8'), 'utf8'),
    ('utf16le-bom', b'\xff\xfe'+(text*30).encode('utf-16-le'), 'base64'),
    ('utf16le-no-bom', (text*30).encode('utf-16-le'), 'base64'),
    ('utf16be-bom', b'\xfe\xff'+(text*30).encode('utf-16-be'), 'base64'),
    ('utf16be-no-bom', (text*30).encode('utf-16-be'), 'base64'),
    ('utf32le', b'\xff\xfe\x00\x00'+text.encode('utf-32-le'), 'base64'),
    ('utf32be', b'\x00\x00\xfe\xff'+text.encode('utf-32-be'), 'base64'),
    ('windows1252', 'café – £15 €20'.encode('cp1252'), 'base64'),
    ('binary-renamed-txt', random.Random(101).randbytes(100_003), 'base64'),
    ('nul-after-3MiB', b'A'*3_145_728+b'\x00END', 'base64'),
    ('invalid-after-3MiB', b'A'*3_145_728+b'\xffEND', 'base64'),
    ('incomplete-final-character', b'A'*1_048_580+b'\xe2\x82', 'base64'),
    ('invalid-part-boundary', b'\x80'*1200, 'base64'),
]


def split(source: Path, output: Path, mode: str | None, maximum: int):
    args = [cli, 'split', '--source', str(source), '--output', str(output),
            '--max-bytes', str(maximum), '--quiet']
    if mode is not None:
        args += ['--encoding', mode]
    return subprocess.run(args, text=True, encoding='utf-8', capture_output=True)


for name, data, wanted in cases:
    with tempfile.TemporaryDirectory() as tmp:
        folder = Path(tmp)
        source = folder/(name+'.txt')  # Extension intentionally cannot determine encoding.
        source.write_bytes(data)
        maximum = 1_100_000 if len(data) > 1_000_000 else 8500
        cp = split(source, folder, 'auto', maximum)
        assert cp.returncode == 0, cp.stderr
        result = json.loads(cp.stdout)
        assert result['Encoding'] == wanted, result
        assert result['SHA256'] == hashlib.sha256(data).hexdigest()
        parts = joiner.inventory(Path(result['Folder']))
        for p in parts:
            raw = p['path'].read_bytes()
            assert len(raw) <= maximum
            assert b'\x00' not in raw
            raw.decode('utf-8', errors='strict')
            assert p['metadata']['encoding'] == wanted
            assert p['metadata']['version'] == 1
        verified = joiner.consume(parts, quiet=True)
        assert verified['sha256'] == hashlib.sha256(data).hexdigest()
        restored = folder/'python-reconstructed.bin'
        joiner.join(parts, restored, quiet=True)
        assert restored.read_bytes() == data
        assert source.read_bytes() == data
        checks.append(f'PASS: Auto {name} -> {wanted}; Python byte-exact join; source unchanged; size cap honoured')

# The CLI default is Auto, not just the GUI default.
for name, data, wanted in (cases[1], cases[4], cases[11]):
    with tempfile.TemporaryDirectory() as tmp:
        folder = Path(tmp)
        source = folder/'default.txt'
        source.write_bytes(data)
        cp = split(source, folder, None, 20000)
        assert cp.returncode == 0, cp.stderr
        assert json.loads(cp.stdout)['Encoding'] == wanted
        checks.append(f'PASS: omitted --encoding defaults to Auto for {name}')

# Explicit readable-only mode must reject rather than silently alter the source.
for name, data in [('nul', b'A\x00B'), ('invalid', b'\xff'),
                   ('incomplete', b'\xe2\x82'), ('bad-boundary', b'\x80'*1000)]:
    with tempfile.TemporaryDirectory() as tmp:
        folder = Path(tmp)
        source = folder/'strict.txt'
        source.write_bytes(data)
        cp = split(source, folder, 'utf8', 8500)
        assert cp.returncode != 0
        assert 'Choose Auto or Encoded text' in cp.stderr
        assert list(folder.iterdir()) == [source]
        assert source.read_bytes() == data
        checks.append(f'PASS: strict readable mode rejects {name}; actionable error; no output or byte changes')

print('\n'.join(checks))
print(f'PASS: {len(checks)} Auto-mode / CLI / Python regression scenarios')
