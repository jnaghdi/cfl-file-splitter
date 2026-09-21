"""25 original synthetic Go/Python interoperability and rejection scenarios."""
import base64, hashlib, importlib.machinery, importlib.util, json, os, pathlib, random, shutil, subprocess, tempfile
from test_support import ROOT, HELPER, native_cli
helper=HELPER
loader=importlib.machinery.SourceFileLoader('cfl_join',str(helper))
spec=importlib.util.spec_from_loader(loader.name,loader); mod=importlib.util.module_from_spec(spec);loader.exec_module(mod)
cli=native_cli()
checks=[]
def cli_split(d, enc, maxsize=20_000_000):
    cp=subprocess.run([cli,'split','--source',str(d/'source.bin'),'--output',str(d),'--encoding',enc,'--max-bytes',str(maxsize),'--quiet'],capture_output=True,text=True,check=True)
    return pathlib.Path(json.loads(cp.stdout)['Folder'])
for enc in ('binary','base64','utf8'):
    sizes=[0,1,17,65537,2_100_003] if enc!='utf8' else [0,1,1000,150000]
    for size in sizes:
        with tempfile.TemporaryDirectory() as temp:
            d=pathlib.Path(temp)
            data=random.Random(size).randbytes(size) if enc!='utf8' else ('Hello\r\nسلام 😀 café\n' * size).encode('utf8')
            (d/'source.bin').write_bytes(data)
            folder=cli_split(d,enc,1_100_000)
            pieces=mod.inventory(folder)
            result=mod.consume(pieces,quiet=True)
            assert result['sha256']==hashlib.sha256(data).hexdigest()
            out=d/'python-joined.bin';mod.join(pieces,out,quiet=True)
            assert out.read_bytes()==data
            checks.append(f'PASS: Go -> Python {enc}, {len(data):,} original bytes')
# Independently create format fixtures in Python; verify and join using Go.
for enc in ('binary','base64','utf8'):
    with tempfile.TemporaryDirectory() as temp:
        d=pathlib.Path(temp);data=('Python-created independent fixture. سلام 😀\r\n'*51).encode('utf8')
        # Use line-aligned chunks for UTF8, arbitrary bytes for other modes.
        chunks=[line for line in data.splitlines(keepends=True)] if enc=='utf8' else [data[i:i+301] for i in range(0,len(data),301)]
        offset=0
        for i,chunk in enumerate(chunks):
            m=dict(format='CFLSPLIT',version=1,set_id='ab'*16,original_name='independent.txt',original_size=len(data),original_sha256=hashlib.sha256(data).hexdigest(),original_modified_utc='2026-09-18T00:00:00Z',created_utc='2026-09-18T00:00:00Z',encoding=enc,part_index=i+1,total_parts=len(chunks),offset=offset,payload_size=len(chunk),payload_sha256=hashlib.sha256(chunk).hexdigest())
            b=base64.b64encode(chunk) if enc=='base64' else chunk
            (d/f'renamed-{len(chunks)-i}.txt').write_bytes(mod.MAGIC+json.dumps(m,ensure_ascii=False).encode()+b'\n\n'+b);offset+=len(chunk)
        out=d/'out.bin'
        subprocess.run([cli,'join','--parts',str(d),'--output',str(out),'--quiet'],capture_output=True,check=True)
        assert out.read_bytes()==data
        checks.append(f'PASS: Python-created {enc} -> Go rejoin')
# Error-path checks against the actual Python helper.
for kind in ('missing','duplicate','corrupt','truncated','mixed','unsafe_name','whole_hash','overwrite'):
    with tempfile.TemporaryDirectory() as temp:
        d=pathlib.Path(temp);data=b'abcdefgh\n'*200
        (d/'source.bin').write_bytes(data);folder=cli_split(d,'base64',8500);p=mod.inventory(folder)
        if kind=='missing':p[0]['path'].unlink()
        elif kind=='duplicate':shutil.copy(p[0]['path'],folder/'duplicate.txt')
        elif kind=='corrupt':
            b=bytearray(p[0]['path'].read_bytes());b[p[0]['start']]=ord('Z');p[0]['path'].write_bytes(b)
        elif kind=='truncated':p[0]['path'].write_bytes(p[0]['path'].read_bytes()[:-1])
        elif kind in ('mixed','unsafe_name','whole_hash'):
            for part in (p if kind=='whole_hash' else p[:1]):
                m=part['metadata'].copy()
                if kind=='mixed':m['set_id']='ff'*16
                elif kind=='unsafe_name':m['original_name']='../../unsafe'
                else:m['original_sha256']='0'*64
                b=part['path'].read_bytes()[part['start']:]
                part['path'].write_bytes(mod.MAGIC+json.dumps(m).encode()+b'\n\n'+b)
        output=d/'new.bin'
        if kind=='overwrite':output.write_bytes(b'KEEP')
        try:mod.join(mod.inventory(folder),output,quiet=True)
        except (mod.SplitError,OSError):pass
        else:raise AssertionError(f'Python accepted {kind}')
        assert output.read_bytes()==b'KEEP' if kind=='overwrite' else not output.exists()
        checks.append(f'PASS: Python rejects {kind}, no completed bad output')
print('\n'.join(checks))
print(f'PASS: {len(checks)} cross-language / Python scenarios')
