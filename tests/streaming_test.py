"""Synthetic 256 MiB + 17 byte streaming test; uses temporary files only."""
import pathlib, tempfile, subprocess, json, hashlib, time, sys
from test_support import HELPER, native_cli
cli=native_cli()
helper=str(HELPER)
start=time.monotonic()
with tempfile.TemporaryDirectory(prefix='cfl-large-') as tmp:
    d=pathlib.Path(tmp);src=d/'sparse_256MiB_plus_17.bin'
    with src.open('wb') as f:
        f.truncate(256*1024*1024+17)
        f.seek(0);f.write(b'CFL LARGE FILE TEST\n')
        f.seek(128*1024*1024-5);f.write(b'MIDDLE_NONZERO')
        f.seek(-17,2);f.write(b'END-MARKER-123456')
    cp=subprocess.run([cli,'split','--source',str(src),'--output',str(d),'--encoding','base64','--max-bytes','20000000','--quiet'],capture_output=True,text=True,check=True)
    result=json.loads(cp.stdout);folder=pathlib.Path(result['Folder'])
    assert result['Parts']==18
    parts=list(folder.glob('part-*.txt'));assert len(parts)==18
    assert all(p.stat().st_size<=20_000_000 for p in parts)
    cp=subprocess.run([sys.executable,helper,'verify','--parts',str(folder),'--quiet'],capture_output=True,text=True,check=True)
    assert json.loads(cp.stdout)['sha256']==result['SHA256']
    out=d/'restored.bin'
    subprocess.run([cli,'join','--parts',str(folder),'--output',str(out),'--quiet'],capture_output=True,check=True)
    def h(p):
        digest=hashlib.sha256()
        with p.open('rb') as f:
            while b:=f.read(1024*1024):digest.update(b)
        return digest.hexdigest()
    assert src.stat().st_size==out.stat().st_size==268435473
    assert h(src)==h(out)==result['SHA256']
    print('PASS: 268,435,473-byte binary source -> 18 encoded TXT pieces, each <= 20,000,000 bytes.')
    print('PASS: independent Python verification of every part and the whole original hash.')
    print('PASS: Go rejoin, read-back verification, exact size and independently computed original/output SHA-256 match.')
    print('SHA256:',result['SHA256'])
    print('This is a synthetic streaming test, not a Windows UI test or a multi-terabyte qualification.')
    print('Test execution seconds:',round(time.monotonic()-start,2))
