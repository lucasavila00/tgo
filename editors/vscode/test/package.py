#!/usr/bin/env python3
import json
import stat
import struct
import sys
import zipfile


def main() -> None:
    archive = sys.argv[1]
    with zipfile.ZipFile(archive) as package:
        names = set(package.namelist())
        icon = "images/icon.png"
        language_icons = {
            "light": "./images/language-light.svg",
            "dark": "./images/language-dark.svg",
        }
        manifest = json.loads(package.read("extension/package.json"))
        if manifest.get("icon") != icon:
            raise SystemExit("VSIX manifest does not declare the TGo icon")
        if f"extension/{icon}" not in names:
            raise SystemExit("VSIX does not contain the TGo icon")
        icon_data = package.read(f"extension/{icon}")
        if icon_data[:8] != b"\x89PNG\r\n\x1a\n":
            raise SystemExit("TGo Marketplace icon is not a PNG")
        if struct.unpack(">II", icon_data[16:24]) != (256, 256):
            raise SystemExit("TGo Marketplace icon is not 256 by 256 pixels")
        if manifest["contributes"]["languages"][0].get("icon") != language_icons:
            raise SystemExit("VSIX manifest does not declare the TGo language icons")
        for language_icon in language_icons.values():
            path = language_icon.removeprefix("./")
            if f"extension/{path}" not in names:
                raise SystemExit(f"VSIX does not contain {path}")
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
