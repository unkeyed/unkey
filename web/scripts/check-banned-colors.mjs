#!/usr/bin/env node
// check-banned-colors.mjs
//
// Tripwire for the color-token consolidation. Tailwind silently drops unknown
// utility classes and CSS `var(--missing)` resolves to nothing, so once a token
// is deleted a stray `bg-gray-600` or `hsl(var(--amber-2))` fails SILENTLY —
// wrong color, no error. This script turns those into a CI failure.
//
// The surviving token system (post-consolidation):
//   - Radix-style numbered scales, steps 1..12 ONLY:
//       gray, grayA, accent, error/errorA, warning/warningA, success/successA,
//       info/infoA, feature/featureA, orange/orangeA, base, blackA, whiteA.
//     Utilities: `bg-gray-2`, `text-warning-11`, `border-accent-6`, ... (1..12).
//   - Structural single-tokens: `border-border`, `ring-ring`.
//   - Chart series vars: `--chart-*` (e.g. `--chart-activity`, `--chart-selection`).
// Everything else in the old vocabulary is gone: the Tailwind numeric palette
// (50..950), the deleted Radix scales (blue/grass/cyan/bronze/brandA/amber),
// the shadcn-style semantic aliases (background/content/brand/warn/alert/subtle/
// muted/card/popover/destructive/foreground), and the `hsl(var(--x))` idiom
// (tokens are full-value OKLCH now, consumed directly or via a scale utility).
//
// Scope: source files (.ts .tsx .css) under web/apps/dashboard and
// web/internal/ui. node_modules/.next/dist are skipped. web/apps/portal is out
// of scope entirely (not scanned).
//
// Allowlist: put `banned-colors-ok` in a comment on the SAME line to suppress a
// deliberate exception (works for both `//` and `/* */`). Use sparingly and
// leave a reason. A file-level allowlist lives in ALLOWLIST_FILES below.

import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const WEB_ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const REPO_ROOT = resolve(WEB_ROOT, "..");

const SCAN_DIRS = ["apps/dashboard", "internal/ui"].map((d) => join(WEB_ROOT, d));
const EXT = new Set([".ts", ".tsx", ".css"]);
const SKIP_DIRS = new Set(["node_modules", ".next", "dist", ".turbo", "coverage"]);

// Files intentionally exempt (relative to repo root). Keep this empty unless a
// file legitimately must reference an old name (e.g. a codemod/migration doc).
const ALLOWLIST_FILES = new Set([]);

const INLINE_OK = "banned-colors-ok";

// --- pattern fragments -------------------------------------------------------

// Tailwind's numeric palette values. Radix steps are 1..12, so matching the
// EXACT Tailwind set (never 11/12) is what keeps `bg-gray-12` from tripping.
const TW_NUM = "50|100|200|300|400|500|600|700|800|900|950";
const TW_UTIL = "bg|text|border|ring|fill|stroke|from|to|via|shadow|divide|outline|decoration";
const TW_COLOR = "gray|red|amber|blue|green|stone|slate|zinc|neutral";

// Deleted Radix scales. `A?` catches the alpha variant (blueA, amberA, ...).
const DEAD_SCALE = "blue|grass|cyan|bronze|brandA|amber";

// Deleted CSS vars. `--chart-activity` survives; bare `--activity` does not, so
// each name is anchored directly to `--`.
const DEAD_VAR = "blueA?|grassA?|cyanA?|bronzeA?|brandA|amberA?|activity";

// Deleted semantic aliases. `warn` must not swallow the surviving `warning`
// scale, and `border`/`ring` are NOT in this list so `border-border`/`ring-ring`
// can never match. The trailing \b + [a-z-] guard handles that (see PATTERNS).
const DEAD_SEM =
  "background|content|brand|warn|alert|subtle|muted|card|popover|destructive|foreground";
const SEM_UTIL = "bg|text|border|ring";

// Leading guard so utility prefixes are not matched inside larger identifiers
// (e.g. the `to` in `photo-...`). Trailing \b lets opacity modifiers (`/50`) and
// quotes terminate a match cleanly.
const LEAD = "(?<![A-Za-z0-9])";

const PATTERNS = [
  {
    id: "hsl-var",
    re: /hsla?\(\s*var\(\s*--/g,
    hint: "dead `hsl(var(--x))` idiom — tokens are full-value OKLCH now; consume the var directly (`var(--x)`) or use a scale utility.",
  },
  {
    id: "tw-numeric",
    re: new RegExp(`${LEAD}(?:${TW_UTIL})-(?:${TW_COLOR})-(?:${TW_NUM})\\b`, "g"),
    hint: "old Tailwind palette naming — use a Radix step 1..12 (e.g. `gray-600` -> `gray-9`, `red-500` -> `error-9`).",
  },
  {
    id: "dead-scale",
    re: new RegExp(`${LEAD}(?:${DEAD_SCALE})A?-[0-9]{1,2}\\b`, "g"),
    hint: "deleted color scale — blue/grass/cyan/bronze/brandA/amber were removed; map to a survivor (accent, info, success, warning, error, feature, orange).",
  },
  {
    id: "dead-var",
    re: new RegExp(`--(?:${DEAD_VAR})\\b`, "g"),
    hint: "deleted CSS variable — bare --blue/--amber/--activity/... removed (note: `--chart-activity` survives, `--activity` does not).",
  },
  {
    id: "dead-semantic",
    re: new RegExp(`${LEAD}(?:${SEM_UTIL})-(?:${DEAD_SEM})(?:-[a-z-]+)?\\b`, "g"),
    hint: "deleted semantic class — background/content/brand/warn/alert/subtle/muted/card/popover/destructive/foreground removed; use a scale step (survivors: `border-border`, `ring-ring`).",
  },
];

// --- walk + scan -------------------------------------------------------------

function* walk(dir) {
  let entries;
  try {
    entries = readdirSync(dir, { withFileTypes: true });
  } catch {
    return;
  }
  for (const e of entries) {
    if (e.name.startsWith(".") && e.name !== ".") {
      if (SKIP_DIRS.has(e.name)) continue;
    }
    const full = join(dir, e.name);
    if (e.isDirectory()) {
      if (SKIP_DIRS.has(e.name)) continue;
      yield* walk(full);
    } else if (e.isFile()) {
      const dot = e.name.lastIndexOf(".");
      if (dot !== -1 && EXT.has(e.name.slice(dot))) yield full;
    }
  }
}

const hits = [];

for (const root of SCAN_DIRS) {
  let ok = false;
  try {
    ok = statSync(root).isDirectory();
  } catch {
    ok = false;
  }
  if (!ok) continue;

  for (const file of walk(root)) {
    const rel = relative(REPO_ROOT, file);
    if (ALLOWLIST_FILES.has(rel)) continue;

    const lines = readFileSync(file, "utf8").split("\n");
    lines.forEach((line, i) => {
      if (line.includes(INLINE_OK)) return;
      for (const p of PATTERNS) {
        p.re.lastIndex = 0;
        let m;
        while ((m = p.re.exec(line)) !== null) {
          hits.push({
            file: rel,
            line: i + 1,
            col: m.index + 1,
            match: m[0],
            id: p.id,
            hint: p.hint,
          });
          if (m.index === p.re.lastIndex) p.re.lastIndex++;
        }
      }
    });
  }
}

// --- report ------------------------------------------------------------------

if (hits.length === 0) {
  console.log("check-banned-colors: no banned color tokens found.");
  process.exit(0);
}

const byId = new Map();
for (const h of hits) {
  if (!byId.has(h.id)) byId.set(h.id, []);
  byId.get(h.id).push(h);
}

console.error(`check-banned-colors: found ${hits.length} banned color token(s).\n`);
for (const p of PATTERNS) {
  const group = byId.get(p.id);
  if (!group || group.length === 0) continue;
  console.error(`[${p.id}] ${p.hint}`);
  for (const h of group) {
    console.error(`  ${h.file}:${h.line}:${h.col}  ${h.match}`);
  }
  console.error("");
}
console.error(
  "Suppress a deliberate exception with a `banned-colors-ok` comment on the same line.",
);
process.exit(1);
