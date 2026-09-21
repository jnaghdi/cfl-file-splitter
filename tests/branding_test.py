"""MIT/name/website/CLI regressions for the Windows 1.0.2 refresh."""
from __future__ import annotations
import hashlib
import json
from pathlib import Path
import re
import subprocess
import tempfile
import unittest
from test_support import ROOT, HELPER, native_cli

NAME = 'CFL FileSplitter For Uploading Large Files To Claude'
EXE = NAME + '.exe'

class BrandingTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.cli = native_cli()
        cls.version = (ROOT/'VERSION').read_text().strip()

    def run_cli(self, *args):
        return subprocess.run([self.cli, *args], capture_output=True, text=True, encoding='utf-8', check=True)

    def test_readme_exact_heading_and_websites(self):
        readme=(ROOT/'README.md').read_text()
        self.assertEqual(readme.splitlines()[0], '# '+NAME)
        for url in ('https://cflab.uk','https://e-discovery.uk'):
            self.assertIn('('+url+')',readme)
        self.assertIn('[MIT](LICENSE)',readme)
        self.assertNotIn('No public open-source licence has been selected',readme)

    def test_cli_version_exact_name(self):
        self.assertEqual(self.run_cli('--version').stdout.strip(), NAME+' '+self.version)

    def test_cli_full_license(self):
        license_text=(ROOT/'LICENSE').read_text()
        self.assertEqual(self.run_cli('--license').stdout,license_text)
        self.assertEqual(self.run_cli('--licence').stdout,license_text)

    def test_cli_help_websites(self):
        text=self.run_cli('--help').stdout
        for expected in (NAME,'https://cflab.uk','https://e-discovery.uk','--license'):
            self.assertIn(expected,text)

    def test_embedded_and_standalone_licensing(self):
        self.assertEqual((ROOT/'LICENSE').read_bytes(),(ROOT/'internal/core/assets/LICENSE.txt').read_bytes())
        helper=HELPER.read_bytes()
        self.assertEqual(helper,(ROOT/'internal/core/assets/CLAUDE_JOINER.txt').read_bytes())
        for p in (ROOT/'examples').glob('*/Parts/CLAUDE_JOINER.txt'):
            self.assertEqual(helper,p.read_bytes())
        for line in (ROOT/'LICENSE').read_text().strip().splitlines():
            if line: self.assertIn('# '+line,helper.decode('utf-8'))

    def test_build_and_ui_names(self):
        for rel in ('scripts/build.sh','scripts/build.ps1','scripts/package_release.py'):
            text=(ROOT/rel).read_text()
            self.assertIn(EXE,text)
            self.assertNotIn('CFL_File_Splitter.exe',text)
        ui=(ROOT/'cmd/cflsplit/main_windows.go').read_text()
        self.assertIn('"STATIC", core.AppName',ui)
        self.assertRegex(ui, r'core\.AppName\s*\+\s*" - v"\s*\+\s*core\.Version')
        self.assertIn('"About & licence"',ui)

    def test_legacy_demo_sets_still_verify(self):
        expected=hashlib.sha256((ROOT/'examples/demo_log.txt').read_bytes()).hexdigest()
        for mode in ('Encoded_TXT','Readable_UTF8'):
            result=json.loads(self.run_cli('verify','--parts',str(ROOT/'examples'/mode/'Parts'),'--quiet').stdout)
            self.assertEqual(result['SHA256'],expected)

    def test_new_split_handover_and_nul_roundtrip(self):
        with tempfile.TemporaryDirectory() as t:
            root=Path(t);source=root/'fictional_utf16.txt'
            original=b'\xff\xfe'+('Fictional text, not case material.\r\n'*500).encode('utf-16-le')
            source.write_bytes(original)
            result=json.loads(self.run_cli('split','--source',str(source),'--output',str(root),'--max-bytes','20000','--encoding','auto','--quiet').stdout)
            self.assertEqual(result['Encoding'],'base64')
            parts=Path(result['Folder'])
            handover=(parts/'CLAUDE_INSTRUCTIONS.txt').read_text()
            for s in (NAME,'https://cflab.uk','https://e-discovery.uk','Software licence: MIT'):
                self.assertIn(s,handover)
            self.assertEqual((parts/'CLAUDE_JOINER.txt').read_bytes(),HELPER.read_bytes())
            dest=root/'NEW_rejoined.txt'
            result=json.loads(self.run_cli('join','--parts',str(parts),'--output',str(dest),'--quiet').stdout)
            self.assertEqual(dest.read_bytes(),original)
            self.assertEqual(source.read_bytes(),original)
            self.assertEqual(result['SHA256'],hashlib.sha256(original).hexdigest())

    def test_no_pending_licensing_in_current_guides(self):
        for rel in ('README.md','LICENSE','CONTRIBUTING.md','SECURITY.md','docs/GITHUB_SETUP.md'):
            text=(ROOT/rel).read_text().lower()
            for outdated in ('licensing status: not selected','no public licence selected','pending licensing review','no public open-source licence has been selected'):
                self.assertNotIn(outdated,text,rel)

if __name__=='__main__':unittest.main(verbosity=2)
