#!/usr/bin/env python3
"""Subset Geist and Geist Mono into app/fonts/. Run by hand and commit the output.

  python3 -m venv /tmp/subset-fonts
  /tmp/subset-fonts/bin/pip install fonttools==4.68.0 brotli==1.2.1
  mise exec -- /tmp/subset-fonts/bin/python -I web/apps/dashboard/scripts/subset-fonts.py

Rerun after any of these:
  - the `geist` package version changes
  - the UI uses a weight below 400 (font-light, font-thin, Radix weight="light")
  - the UI turns on an OpenType feature not in FEATURES (slashed-zero,
    diagonal-fractions, a stylistic set) or sets a `lang` other than English
  - a character outside CORE starts showing up on most pages
"""

import json
import subprocess
import unicodedata
from io import BytesIO
from pathlib import Path

from fontTools import subset
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer

DASHBOARD = Path(__file__).resolve().parent.parent
SOURCE = DASHBOARD / "node_modules/geist/dist/fonts"
OUT = DASHBOARD / "app/fonts"

# The UI renders 400-700, but cutting the top below the 900 master makes
# fontTools round new deltas, which moves 500-700 outlines by up to half a unit.
# Cutting at the 400 default master changes nothing.
WEIGHTS = (400, 900)

# Browser defaults plus tnum for tabular-nums. locl is left out because Geist
# only defines it for AZE/CRT/KAZ/MAH/MOL/NLD/ROM/TAT/TRK and the UI is lang="en".
FEATURES = ["ccmp", "kern", "liga", "mark", "mkmk", "tnum"]


def codepoints(*items):
    out = set()
    for item in items:
        if isinstance(item, tuple):
            out.update(range(item[0], item[1] + 1))
        else:
            out.add(item)
    return out


CORE = codepoints(
    (0x20, 0x7E),
    (0xA0, 0x17F),
    (0x300, 0x36F),
    0x2013,
    0x2014,
    0x2018,
    0x2019,
    0x201C,
    0x201D,
    0x2022,
    0x2026,
    0x21E7,
)

# Geist has no cmap entry for these, but browsers draw them from its space and
# hyphen glyphs. Leaving them out of every range sends them to a system font.
DRAWN_FROM_CORE = codepoints((0x2000, 0x200A), 0x2010, 0x2011, 0x2028, 0x2029, 0x202F, 0x205F)

FONTS = [
    {
        "file": "geist-sans",
        "source": SOURCE / "geist-sans/Geist-Variable.woff2",
        "export": "GeistSans",
        "options": {"variable": "--font-geist-sans"},
    },
    {
        "file": "geist-mono",
        "source": SOURCE / "geist-mono/GeistMono-Variable.woff2",
        "export": "GeistMono",
        "options": {
            "variable": "--font-geist-mono",
            "adjustFontFallback": False,
            "fallback": [
                "ui-monospace",
                "SFMono-Regular",
                "Roboto Mono",
                "Menlo",
                "Monaco",
                "Liberation Mono",
                "DejaVu Sans Mono",
                "Courier New",
                "monospace",
            ],
        },
    },
]


def unicode_range(cps):
    runs = []
    for cp in sorted(cps):
        if runs and cp == runs[-1][1] + 1:
            runs[-1][1] = cp
        else:
            runs.append([cp, cp])
    return ", ".join(f"U+{a:04X}" if a == b else f"U+{a:04X}-{b:04X}" for a, b in runs)


def decomposable(available):
    """Characters Geist lacks but browsers draw from it as base letter plus mark.

    Chrome only shapes with characters inside a face's unicode-range, so the
    marks join the range of every file that decomposes with them.
    """
    out = {}
    for cp in range(0x110000):
        if cp in available:
            continue
        decomposition = unicodedata.decomposition(chr(cp))
        if not decomposition or decomposition.startswith("<"):
            continue
        nfd = [ord(c) for c in unicodedata.normalize("NFD", chr(cp))]
        if all(c in available for c in nfd):
            out[cp] = nfd
    return out


def subset_to(instanced, cps, path):
    options = subset.Options()
    options.flavor = "woff2"
    options.layout_features = FEATURES
    options.hinting = False
    options.glyph_names = False
    options.name_legacy = False
    font = TTFont(BytesIO(instanced), recalcTimestamp=False)
    subsetter = subset.Subsetter(options)
    subsetter.populate(unicodes=cps)
    subsetter.subset(font)
    font.save(path)
    assert set(TTFont(path).getBestCmap()) == cps, f"{path.name}: cmap differs from request"


def build(spec):
    font = instancer.instantiateVariableFont(
        TTFont(spec["source"], recalcTimestamp=False), {"wght": WEIGHTS}, updateFontNames=False
    )
    buffer = BytesIO()
    font.save(buffer)
    available = set(font.getBestCmap())
    assert {0x20, 0x2D} <= CORE, "DRAWN_FROM_CORE needs space and hyphen in the core"

    owned = {}
    claimed = set()
    for part, wanted in (("core", CORE), ("rest", available)):
        owned[part] = (wanted & available) - claimed
        claimed |= owned[part]

    cmaps = {part: set(cps) for part, cps in owned.items()}
    ranges = {part: set(cps) for part, cps in owned.items()}
    ranges["core"] |= DRAWN_FROM_CORE
    for cp, nfd in decomposable(available).items():
        part = next(p for p, cps in owned.items() if nfd[0] in cps)
        ranges[part].add(cp)
        ranges[part].update(nfd)
        cmaps[part].update(nfd)

    files = {}
    for part in owned:
        name = f"{spec['file']}-{part}.woff2"
        subset_to(buffer.getvalue(), cmaps[part], OUT / name)
        files[part] = (name, unicode_range(ranges[part]))
    return files


def font_face(family, name, ranges):
    return (
        "@font-face {\n"
        f"  font-family: {family};\n"
        f'  src: url("./{name}") format("woff2");\n'
        "  font-display: swap;\n"
        f"  font-weight: {WEIGHTS[0]} {WEIGHTS[1]};\n"
        f"  unicode-range: {ranges};\n"
        "}\n"
    )


def main():
    OUT.mkdir(exist_ok=True)
    ts = ['import localFont from "next/font/local";']
    css = []
    for spec in FONTS:
        files = build(spec)
        core, core_ranges = files["core"]
        options = {
            "src": f"./{core}",
            "weight": f"{WEIGHTS[0]} {WEIGHTS[1]}",
            **spec["options"],
            "declarations": [
                {"prop": "font-family", "value": spec["export"]},
                {"prop": "unicode-range", "value": core_ranges},
            ],
        }
        ts.append(f"export const {spec['export']} = localFont({json.dumps(options, indent=2)});")
        css.append(font_face(spec["export"], *files["rest"]))
    (OUT / "geist.ts").write_text("\n\n".join(ts) + "\n")
    (OUT / "geist.css").write_text("\n".join(css))
    subprocess.run(
        ["pnpm", "biome", "format", "--write", "apps/dashboard/app/fonts"],
        cwd=DASHBOARD.parent.parent,
        check=True,
    )
    for path in sorted(OUT.glob("*.woff2")):
        print(f"{path.stat().st_size:>7}  {path.name}")


if __name__ == "__main__":
    main()
