"""Execute publication shell steps with synthetic artifacts and a fake gh client."""

import os
from pathlib import Path
import subprocess
import sys
import tempfile
import textwrap
import unittest

WORKFLOW = Path(__file__).resolve().parents[2] / ".github/workflows/release.yml"


def step_script(name):
    step = WORKFLOW.read_text().split(f"      - name: {name}\n", 1)[1]
    body = step.split("        run: |\n", 1)[1]
    lines = []
    for line in body.splitlines():
        if line and not line.startswith("          "):
            break
        lines.append(line[10:])
    return "\n".join(lines)


class PublicationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        (self.root / "dist").mkdir()
        for name in ["checksums.txt", "rekal_linux_amd64.tar.gz", "rekal_linux_arm64.tar.gz", "rekal_darwin_arm64.tar.gz"]:
            (self.root / "dist" / name).write_text("synthetic")
        fake = self.root / "gh"
        fake.write_text(f"#!{sys.executable}\n" + textwrap.dedent('''\
            import json, os, sys
            from pathlib import Path
            args = sys.argv[1:]
            with open(os.environ['CALL_LOG'], 'a') as f:
                f.write(json.dumps(args) + '\\n')
            if args[1] == 'view':
                state = os.environ['RELEASE_STATE']
                if state == 'missing': sys.exit(1)
                print(json.dumps({'isDraft': state == 'draft'}))
            if args[1] == 'upload' and os.environ.get('FAIL_UPLOAD'): sys.exit(1)
        '''))
        fake.chmod(0o755)
        self.env = {**os.environ, "PATH": str(self.root) + os.pathsep + os.environ['PATH'],
                    "CALL_LOG": str(self.root / "calls"), "RELEASE_TAG": "v2.0.0", "RELEASE_STATE": "missing"}

    def publish(self, **env):
        script = step_script("Publish complete immutable release")
        script = script.replace('/tmp/release.json', str(self.root / 'release.json'))
        result = subprocess.run(['bash', '-e', '-o', 'pipefail', '-c', script],
                                cwd=self.root, env={**self.env, **env}, capture_output=True, text=True)
        import json
        calls = [json.loads(line) for line in (self.root / 'calls').read_text().splitlines()]
        return result, calls

    def test_new_release_uploads_before_publish(self):
        result, calls = self.publish()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual([c[1] for c in calls], ['view', 'create', 'upload', 'edit'])
        self.assertIn('--draft', calls[1])
        self.assertIn('--verify-tag', calls[1])
        self.assertIn('--draft=false', calls[-1])
        for name in ['checksums.txt', 'rekal_linux_amd64.tar.gz', 'rekal_linux_arm64.tar.gz', 'rekal_darwin_arm64.tar.gz']:
            self.assertIn('dist/' + name, calls[2])

    def test_retry_resumes_only_draft(self):
        result, calls = self.publish(RELEASE_STATE='draft')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual([c[1] for c in calls], ['view', 'upload', 'edit'])

    def test_published_release_is_untouched(self):
        result, calls = self.publish(RELEASE_STATE='published')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual([c[1] for c in calls], ['view'])

    def test_upload_failure_does_not_publish(self):
        result, calls = self.publish(FAIL_UPLOAD='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn('edit', [c[1] for c in calls])

    def test_missing_archive_fails_checksum_gate(self):
        (self.root / 'dist/rekal_linux_arm64.tar.gz').unlink()
        result = subprocess.run(['bash', '-e', '-o', 'pipefail', '-c', step_script('Generate checksums')],
                                cwd=self.root, capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)


if __name__ == '__main__':
    unittest.main()
