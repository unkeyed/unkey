from pathlib import Path
import os
import shutil
import subprocess
import sys
import tempfile
import unittest


class InstallWebTest(unittest.TestCase):
    def test_restored_dependencies(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            subprocess.run(["git", "init", "-q", directory], check=True)
            (root / ".agents").mkdir()
            shutil.copyfile(Path(__file__).with_name("install-web.py"), root / ".agents/install-web.py")
            (root / "web").mkdir()
            (root / "web/package.json").write_text("{}")
            (root / "web/source.ts").write_text("first")
            (root / ".gitignore").write_text("node_modules/\nesm/\n")
            subprocess.run(["git", "add", "."], cwd=root, check=True)
            installer = root / "mise"
            installer.write_text(
                "#!/bin/sh\nset -eu\n"
                'test "$*" = "run --force install-web"\n'
                "echo install >> installs\n"
                "test ! -e fail\n"
                "mkdir -p web/node_modules/.pnpm web/node_modules/.bin web/internal/api/esm\n"
                "touch web/node_modules/.modules.yaml web/node_modules/.pnpm/lock.yaml "
                "web/node_modules/.bin/tsc web/internal/api/esm/index.js web/internal/api/esm/index.d.ts\n"
            )
            installer.chmod(0o755)

            def run(installs, success=True):
                result = subprocess.run(
                    [sys.executable, str(root / ".agents/install-web.py"), str(installer)],
                    cwd=root, capture_output=True, text=True,
                )
                self.assertEqual(result.returncode == 0, success, result.stderr)
                self.assertEqual(len((root / "installs").read_text().splitlines()), installs)

            run(1)
            run(1)
            (root / "unrelated.go").write_text("package example")
            run(1)

            for count, name in enumerate([
                "web/package.json", "web/source.ts", "web/pnpm-lock.yaml",
                "web/.npmrc", ".npmrc", ".mise/config.toml",
            ], start=2):
                path = root / name
                path.parent.mkdir(exist_ok=True)
                path.touch(exist_ok=True)
                stat = path.stat()
                path.write_text("changed")
                os.utime(path, ns=(stat.st_atime_ns, stat.st_mtime_ns))
                run(count)
                run(count)

            (root / "web/source.ts").unlink()
            run(8)
            (root / "web/node_modules/.bin/tsc").unlink()
            run(9)
            (root / "web/internal/api/esm/index.js").unlink()
            run(10)
            (root / "web/package.json").write_text("changed again")
            (root / "fail").touch()
            run(11, success=False)
            self.assertFalse((root / "web/node_modules/.amp-install.sha256").exists())
            (root / "fail").unlink()
            run(12)
            run(12)


if __name__ == "__main__":
    unittest.main()
