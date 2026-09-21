"""Standard-library checks for source inventory, helper sync and byte-exact demos."""
from __future__ import annotations
import hashlib
import importlib.util
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT/'scripts'))
import source_manifest
spec = importlib.util.spec_from_file_location('cfl_repo_joiner', ROOT/'tools/claude_joiner.py')
joiner = importlib.util.module_from_spec(spec)
spec.loader.exec_module(joiner)

class RepositoryTests(unittest.TestCase):
    def test_source_inventory(self):
        self.assertGreater(len(source_manifest.verify()), 40)

    def test_helper_copies_are_identical(self):
        self.assertEqual((ROOT/'tools/claude_joiner.py').read_bytes(),
                         (ROOT/'internal/core/assets/CLAUDE_JOINER.txt').read_bytes())

    def test_application_version_matches(self):
        version = (ROOT/'VERSION').read_text().strip()
        self.assertRegex(version, r'^\d+\.\d+\.\d+$')
        core = (ROOT/'internal/core/core.go').read_text()
        self.assertIn(f'const Version = "{version}"', core)

    def test_demo_sets_reconstruct_exactly(self):
        original = (ROOT/'examples/demo_log.txt').read_bytes()
        for mode in ('Encoded_TXT', 'Readable_UTF8'):
            with self.subTest(mode=mode), tempfile.TemporaryDirectory() as tmp:
                parts = joiner.inventory(ROOT/'examples'/mode/'Parts')
                result = joiner.consume(parts, quiet=True)
                self.assertEqual(result['sha256'], hashlib.sha256(original).hexdigest())
                out = Path(tmp)/'restored.txt'
                joiner.join(parts, out, quiet=True)
                self.assertEqual(out.read_bytes(), original)

    def test_fixture_line_ending_protection(self):
        attributes = (ROOT/'.gitattributes').read_text()
        self.assertIn('examples/** -text', attributes)
        self.assertIn('internal/core/assets/CLAUDE_JOINER.txt -text', attributes)
        self.assertIn('tools/claude_joiner.py -text', attributes)

    def test_workflow_pins_and_token_scope(self):
        for workflow in (ROOT/'.github/workflows').glob('*.yml'):
            text = workflow.read_text()
            for dependency in re.findall(r'uses:\s*(\S+)', text):
                self.assertRegex(dependency, r'^[\w-]+/[\w-]+@[a-f0-9]{40}$')
            self.assertNotIn('pull_request_target:', text)
            self.assertIn('persist-credentials: false', text)
        ci = (ROOT/'.github/workflows/ci.yml').read_text()
        self.assertNotIn('contents: write', ci)
        release = (ROOT/'.github/workflows/release.yml').read_text()
        self.assertIn('--draft', release)
        self.assertIn('--verify-tag', release)

    def test_publisher_has_private_only_remote_creation(self):
        text = (ROOT/'scripts/publish-github.ps1').read_text()
        self.assertIn("'--private'", text)
        self.assertNotIn("'--public'", text)
        self.assertNotIn("'--force'", text)
        self.assertIn("-cne 'CREATE'", text)
        self.assertIn('Get-FileHash', text)

    def test_required_source_files(self):
        for relative in ('README.md','LICENSE','go.mod','VERSION','SECURITY.md',
                         'scripts/build.ps1','scripts/build.sh','scripts/test.ps1',
                         'scripts/test.sh','scripts/package_release.py',
                         'scripts/publish-github.ps1','docs/GITHUB_SETUP.md',
                         'docs/FORMAT.md','.github/workflows/ci.yml',
                         '.github/workflows/release.yml'):
            with self.subTest(path=relative):
                self.assertTrue((ROOT/relative).is_file())

    def test_no_binaries_or_case_files_in_inventory(self):
        blocked = {'.exe','.dll','.zip','.e01','.ex01','.ufdr','.pst','.ost',
                   '.vhd','.vhdx','.vmdk','.dd','.img','.pem','.pfx','.p12'}
        for _, relative in source_manifest.entries():
            self.assertNotIn(Path(relative).suffix.lower(), blocked)

    def test_inventory_rejects_unsafe_paths(self):
        for relative in ('../secret', '/secret', 'docs/../../secret',
                         r'docs\secret', 'bin/app.exe', 'docs/./x',
                         'docs//x', 'C:/secret'):
            with self.subTest(path=relative), self.assertRaises(ValueError):
                source_manifest.safe_path(relative)

if __name__ == '__main__':
    unittest.main(verbosity=2)
