"""Exercise the installer without network access or changing shell profiles."""

import hashlib
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest


INSTALLER = Path(__file__).resolve().parents[1] / "install.sh"


class InstallerTests(unittest.TestCase):
    def test_archive_ownership_is_not_restored(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            archive = root / "rekal_linux_amd64.tar.gz"
            with tarfile.open(archive, "w:gz") as handle:
                for name, content in {
                    "rekal": b"#!/bin/sh\necho 'rekal 1.0.3'\n",
                    "LICENSE": b"Apache-2.0\n",
                    "README.md": b"installer fixture\n",
                }.items():
                    info = tarfile.TarInfo(name)
                    info.uid = info.gid = 1001
                    info.uname = info.gname = "runner"
                    info.mode = 0o755 if name == "rekal" else 0o644
                    info.size = len(content)
                    handle.addfile(info, io.BytesIO(content))
            digest = hashlib.sha256(archive.read_bytes()).hexdigest()
            (root / "checksums.txt").write_text(f"{digest}  {archive.name}\n")
            tools = root / "tools"
            tools.mkdir()
            curl = tools / "curl"
            curl.write_text(
                "#!/usr/bin/env python3\n"
                "import os,pathlib,shutil,sys\n"
                "args=sys.argv[1:]\n"
                "url=next(a for a in args if a.startswith('https://'))\n"
                "dest=args[args.index('-o')+1]\n"
                "shutil.copyfile(pathlib.Path(os.environ['INSTALL_FIXTURES'])/url.rsplit('/',1)[1],dest)\n"
            )
            curl.chmod(0o755)
            target = root / "bin"
            env = {
                **os.environ,
                "PATH": f"{tools}{os.pathsep}{target}{os.pathsep}{os.environ['PATH']}",
                "REKAL_VERSION": "v1.0.3",
                "INSTALL_FIXTURES": str(root),
            }
            env.pop("TAR_OPTIONS", None)
            result = subprocess.run(
                ["bash", str(INSTALLER), "--target", str(target)],
                env=env, capture_output=True, text=True, check=False,
            )
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertEqual(
                subprocess.check_output([str(target / "rekal"), "version"], text=True).strip(),
                "rekal 1.0.3",
            )


if __name__ == "__main__":
    unittest.main()
