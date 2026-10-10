"""Check Markdown line width. Exclude AGENTS.md and the root coordinator goal."""

import os
from pathlib import Path
import re
import subprocess
from urllib.parse import unquote


staged = os.environ.get("MARKDOWN_STAGED") == "1"
command = ["git", "ls-files", "-z", "--cached"]
if not staged:
    command += ["--others", "--exclude-standard"]
paths = subprocess.check_output(command).split(b"\0")
failed = False
link_pattern = re.compile(r"\[[^]]*\]\(([^)]+)\)")

for raw_path in sorted(set(paths) - {b""}):
    path = Path(os.fsdecode(raw_path))
    if path.suffix.lower() not in {".md", ".markdown"}:
        continue
    if staged:
        content = subprocess.check_output(["git", "show", f":{path}"]).decode("utf-8")
    else:
        if not path.is_file():
            continue
        content = path.read_text(encoding="utf-8")
    if "https://github.com/lucasavila00/go2" in content:
        print(f"{path}: link uses the old lucasavila00/go2 repository name")
        failed = True
    for target in link_pattern.findall(content):
        target = target.split("#", 1)[0]
        if not target or "://" in target or target.startswith("mailto:"):
            continue
        linked = path.parent / unquote(target)
        if not linked.exists():
            print(f"{path}: local link target does not exist: {target}")
            failed = True
    # The root coordinator goal must stay on one line.
    if path.name == "AGENTS.md" or path == Path("coordinator.md"):
        continue
    limit = 120
    for number, line in enumerate(content.splitlines(), 1):
        width = len(line.expandtabs(4))
        if width > limit:
            print(f"{path}:{number}: {width} characters; limit is {limit}")
            failed = True

raise SystemExit(1 if failed else 0)
