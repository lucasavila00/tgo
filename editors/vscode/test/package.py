#!/usr/bin/env python3
import stat
import sys
import zipfile


def main() -> None:
    archive = sys.argv[1]
    with zipfile.ZipFile(archive) as package:
        names = set(package.namelist())
        helper = "extension/bin/tgonav"
        if helper not in names:
            raise SystemExit("VSIX does not contain the bundled tgonav helper")
        mode = package.getinfo(helper).external_attr >> 16
        if mode & stat.S_IXUSR == 0:
            raise SystemExit("bundled tgonav helper is not executable")
        forbidden = (
            "extension/node_modules/",
            "extension/test/",
        )
        for name in names:
            if name.startswith(forbidden):
                raise SystemExit(f"VSIX contains development file {name}")
        if "extension/package-lock.json" in names:
            raise SystemExit("VSIX contains package-lock.json")


if __name__ == "__main__":
    main()
