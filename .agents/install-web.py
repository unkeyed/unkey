import hashlib
import os
from pathlib import Path
import platform
import subprocess
import sys


root = Path(__file__).resolve().parent.parent
os.chdir(root)
paths = subprocess.check_output([
    "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard", "--",
    "web", ".mise", ".agents/install-web.py", "dev/install-mise",
]).split(b"\0")
inputs = {os.fsdecode(path) for path in paths if path}
inputs.update([".npmrc", "web/.npmrc", ".pnpmfile.cjs", "web/.pnpmfile.cjs"])
digest = hashlib.sha256()
digest.update(f"{platform.system()}:{platform.machine()}\0".encode())
for name in sorted(inputs):
    path = Path(name)
    digest.update(os.fsencode(name) + b"\0")
    if path.is_file():
        digest.update(hashlib.sha256(path.read_bytes()).digest())
    else:
        digest.update(b"missing")
fingerprint = digest.hexdigest()
marker = Path("web/node_modules/.amp-install.sha256")
outputs = [
    Path("web/node_modules/.modules.yaml"),
    Path("web/node_modules/.pnpm/lock.yaml"),
    Path("web/node_modules/.bin/tsc"),
    Path("web/internal/api/esm/index.js"),
    Path("web/internal/api/esm/index.d.ts"),
]
if marker.is_file() and marker.read_text().strip() == fingerprint and all(p.is_file() for p in outputs):
    print("Web install inputs and outputs match; reusing installed dependencies.", flush=True)
else:
    marker.unlink(missing_ok=True)
    subprocess.run([sys.argv[1], "run", "--force", "install-web"], check=True)
    if not all(p.is_file() for p in outputs):
        sys.exit("Web installation did not produce the expected outputs; cache not saved.")
    marker.write_text(fingerprint + "\n")
