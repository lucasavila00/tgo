"""Check Markdown size. Exclude AGENTS.md."""

import argparse
import os
from pathlib import Path
import subprocess


parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("check", choices=("lines", "width"))
args = parser.parse_args()
staged = os.environ.get("MARKDOWN_STAGED") == "1"
command = ["git", "ls-files", "-z", "--cached"]
if not staged:
    command += ["--others", "--exclude-standard"]
paths = subprocess.check_output(command).split(b"\0")
failed = False

for raw_path in sorted(set(paths) - {b""}):
    path = Path(os.fsdecode(raw_path))
    if path.name == "AGENTS.md" or path.suffix.lower() not in {".md", ".markdown"}:
        continue
    if staged:
        content = subprocess.check_output(["git", "show", f":{path}"]).decode("utf-8")
    else:
        if not path.is_file():
            continue
        content = path.read_text(encoding="utf-8")
    lines = content.splitlines()
    if args.check == "lines":
        if len(lines) > 100:
            print(f"{path}: {len(lines)} lines; limit is 100")
            failed = True
    else:
        for number, line in enumerate(lines, 1):
            width = len(line.expandtabs(4))
            if width > 120:
                print(f"{path}:{number}: {width} characters; limit is 120")
                failed = True

raise SystemExit(1 if failed else 0)
