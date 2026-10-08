"""Check Markdown line width. Exclude AGENTS.md and docs/research/go/."""

import os
from pathlib import Path
import subprocess


staged = os.environ.get("MARKDOWN_STAGED") == "1"
command = ["git", "ls-files", "-z", "--cached"]
if not staged:
    command += ["--others", "--exclude-standard"]
paths = subprocess.check_output(command).split(b"\0")
failed = False
proposal_roots = (Path("docs/tgo"),)

for raw_path in sorted(set(paths) - {b""}):
    path = Path(os.fsdecode(raw_path))
    if path.name == "AGENTS.md" or path.suffix.lower() not in {".md", ".markdown"}:
        continue
    if path.is_relative_to("docs/research/go"):
        continue
    proposal = next((root for root in proposal_roots if path.is_relative_to(root)), None)
    if staged:
        content = subprocess.check_output(["git", "show", f":{path}"]).decode("utf-8")
    else:
        if not path.is_file():
            continue
        content = path.read_text(encoding="utf-8")
    limit = 100 if proposal is not None else 120
    for number, line in enumerate(content.splitlines(), 1):
        width = len(line.expandtabs(4))
        if width > limit:
            print(f"{path}:{number}: {width} characters; limit is {limit}")
            failed = True

raise SystemExit(1 if failed else 0)
