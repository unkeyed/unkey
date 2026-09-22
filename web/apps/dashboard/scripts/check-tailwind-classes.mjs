#!/usr/bin/env node
// Fails when a class-like token used in source compiles to no CSS rule.
//
// Compiles styles/tailwind.css with the tailwindcss version pinned in
// web/pnpm-workspace.yaml, resolved through this app's own node_modules (the
// pnpm store also holds stray newer tailwindcss copies pulled in by other
// packages' devDependencies; requiring "tailwindcss" from here avoids them).
// A token passes only if Tailwind's own source scanner finds the exact same
// string AND building it grows the compiled CSS. Both checks are necessary:
// scanner membership alone would pass a syntactically valid class that the
// scanner failed to extract from its surrounding source (that IS a bug —
// it means production ships without it); a growth check alone would pass a
// valid-looking class fed to it directly even when nothing in the repo
// would ever produce that literal candidate string.
//
// Scans this app, web/internal/ui/src, and web/apps/design, which renders the
// same @unkey/ui token layer and documents the classes the dashboard should
// use. All three compile against this app's styles/tailwind.css.
//
// Run: node web/apps/dashboard/scripts/check-tailwind-classes.mjs
//
// Dynamic class construction (e.g. `` `bg-${color}-500` ``) can't be
// verified statically. Add the literal generated class names this produces
// to tailwind-class-allowlist.json to silence them once you've confirmed by
// hand that they compile.

import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dashboardDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const require = createRequire(path.join(dashboardDir, "package.json"));
const cssEntry = path.join(dashboardDir, "styles/tailwind.css");
const allowlistPath = path.join(dashboardDir, "scripts/tailwind-class-allowlist.json");

const twPkgJsonPath = require.resolve("tailwindcss/package.json");
const twVersion = JSON.parse(fs.readFileSync(twPkgJsonPath, "utf8")).version;
const pnpmStoreDir = path.resolve(path.dirname(twPkgJsonPath), "../../..");

function resolvePinned(pkg, entryFile) {
  const dir = path.join(
    pnpmStoreDir,
    `@tailwindcss+${pkg}@${twVersion}`,
    "node_modules",
    "@tailwindcss",
    pkg,
  );
  const entry = path.join(dir, entryFile);
  if (!fs.existsSync(entry)) {
    throw new Error(
      `Expected @tailwindcss/${pkg}@${twVersion} at ${entry}. The pnpm store layout changed — reinstall or update the version derivation in this script.`,
    );
  }
  return require(entry);
}

const { compile } = resolvePinned("node", "dist/index.js");
const { Scanner } = resolvePinned("oxide", "index.js");

const allowlist = new Set(JSON.parse(fs.readFileSync(allowlistPath, "utf8")));

// Utility classes that compile fine but should never be written by hand.
// Matched against the whole class-like token, exact string only (no
// substring/prefix match), so `--border`, `border-strong`, `border-input`,
// and variant-prefixed forms like `hover:border-border` are unaffected —
// add those as their own entries if they need banning too.
const BANNED_CLASSES = [
  {
    token: "border-border",
    reason:
      "redundant: the base rule in web/internal/ui/theme.css " +
      "already sets border-color. Write bare `border` instead.",
  },
];
const bannedReasonByToken = new Map(BANNED_CLASSES.map(({ token, reason }) => [token, reason]));

// Namespace guard (see the theme.css header): raised, table-header and strong
// must stay in --background-color-*/--border-color-*, never --color-*, because
// Tailwind v4 derives every utility family from a single --color-* entry. Each
// of these must compile to nothing; if one starts emitting CSS, a --color-*
// registration for that token leaked in. Add more tokens here as needed.
const NAMESPACE_GUARD_CLASSES = [
  { token: "border-raised", namespace: "--background-color-raised" },
  { token: "border-table-header", namespace: "--background-color-table-header" },
  { token: "text-raised", namespace: "--background-color-raised" },
  { token: "bg-strong", namespace: "--border-color-strong" },
];

const css = fs.readFileSync(cssEntry, "utf8");
const base = path.dirname(cssEntry);
const result = await compile(css, { base, onDependency: () => {} });

// Mirrors @tailwindcss/postcss: an unset root falls back to the app root
// (its process.cwd() at build time), not the CSS file's own directory.
const rootSource =
  result.root === "none"
    ? []
    : result.root === null
      ? [{ base: dashboardDir, pattern: "**/*", negated: false }]
      : [{ ...result.root, negated: false }];
const designDir = path.resolve(dashboardDir, "../design");
const scanner = new Scanner({
  sources: rootSource.concat(result.sources, [
    { base: designDir, pattern: "**/*", negated: false },
  ]),
});
const scannedCandidates = new Set(scanner.scan());

const IGNORED_DIRS = new Set(["node_modules", ".next", ".turbo", "dist", "build"]);
function collectFiles(dir, out) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.name.startsWith(".") || IGNORED_DIRS.has(entry.name)) {
      continue;
    }
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      collectFiles(full, out);
    } else if (/\.(tsx?|jsx?)$/.test(entry.name)) {
      out.push(full);
    }
  }
  return out;
}

const sourceRoots = [dashboardDir, path.resolve(dashboardDir, "../../internal/ui/src"), designDir];
const files = [...new Set(sourceRoots.flatMap((dir) => collectFiles(dir, [])))];

const CONTEXT_START = /\bclassName\s*=\s*\{|\bclassName\s*=\s*["'`]|\b(?:cn|clsx|cva|tv|cx)\s*\(/g;
const LITERAL = /`(?:\\.|[^`\\])*`|'(?:\\.|[^'\\])*'|"(?:\\.|[^"\\])*"/g;
const TOKEN_CHARS = /^[A-Za-z0-9_\-:./[\]()#%*,!&~>=+]+$/;

function captureBalanced(text, openIdx, open, close) {
  let depth = 0;
  for (let i = openIdx; i < text.length; i++) {
    if (text[i] === open) {
      depth++;
    } else if (text[i] === close) {
      depth--;
      if (depth === 0) {
        return text.slice(openIdx, i + 1);
      }
    }
  }
  return text.slice(openIdx);
}

function splitTopLevel(token, sep) {
  const parts = [];
  let depth = 0;
  let buf = "";
  for (const ch of token) {
    if (ch === "[") {
      depth++;
    } else if (ch === "]") {
      depth--;
    }
    if (ch === sep && depth === 0) {
      parts.push(buf);
      buf = "";
    } else {
      buf += ch;
    }
  }
  parts.push(buf);
  return parts;
}

function isClassLike(token) {
  if (token.length < 2 || token.length > 120) {
    return false;
  }
  if (!TOKEN_CHARS.test(token)) {
    return false;
  }
  if (/^https?:/.test(token) || token.startsWith("./") || token.startsWith("../")) {
    return false;
  }
  if (/\.(tsx?|jsx?|json|css|scss|svg|png|jpe?g|gif|md|mdx|ya?ml)$/i.test(token)) {
    return false;
  }
  let depth = 0;
  for (const ch of token) {
    if (ch === "[") {
      depth++;
    } else if (ch === "]") {
      depth--;
      if (depth < 0) {
        return false;
      }
    }
  }
  if (depth !== 0) {
    return false;
  }
  const parts = splitTopLevel(token, ":");
  let last = parts[parts.length - 1].replace(/^!/, "").replace(/!$/, "");
  last = last.replace(/\/(\[[^\]]+\]|\d+)$/, "");
  const bracketIdx = last.indexOf("[");
  if (bracketIdx !== -1) {
    // prefix-[arbitrary-value] (e.g. bg-[hsl(...)]) or bare [arbitrary-property:value]
    const prefix = last.slice(0, bracketIdx);
    return prefix === "" || /^-?[a-z][a-z0-9]*(-[a-z0-9]*)*-$/i.test(prefix);
  }
  return /^-?[a-z][a-z0-9]*(-[a-z0-9]+)+$/i.test(last);
}

// { token -> Map(file -> Set(line)) }
const found = new Map();
function record(token, file, line) {
  if (!found.has(token)) {
    found.set(token, new Map());
  }
  const byFile = found.get(token);
  if (!byFile.has(file)) {
    byFile.set(file, new Set());
  }
  byFile.get(file).add(line);
}

for (const file of files) {
  const text = fs.readFileSync(file, "utf8");
  const rel = path.relative(dashboardDir, file);
  CONTEXT_START.lastIndex = 0;
  let m = CONTEXT_START.exec(text);
  while (m !== null) {
    const matched = m[0];
    const openIdx = m.index + matched.length - 1;
    let span;
    if (matched.endsWith("{")) {
      span = captureBalanced(text, openIdx, "{", "}");
    } else if (matched.endsWith("(")) {
      span = captureBalanced(text, openIdx, "(", ")");
    } else {
      const quote = text[openIdx];
      let end = -1;
      for (let i = openIdx + 1; i < text.length; i++) {
        if (text[i] === "\\") {
          i++;
          continue;
        }
        if (text[i] === quote) {
          end = i;
          break;
        }
      }
      span = end === -1 ? "" : text.slice(openIdx, end + 1);
    }
    LITERAL.lastIndex = 0;
    let lm = LITERAL.exec(span);
    while (lm !== null) {
      const inner = lm[0].slice(1, -1).replace(/\$\{[^}]*\}/g, "\u0000");
      const line = text.slice(0, m.index + LITERAL.lastIndex).split("\n").length;
      for (const word of inner.split(/\s+/)) {
        if (word && isClassLike(word)) {
          record(word, rel, line);
        }
      }
      lm = LITERAL.exec(span);
    }
    m = CONTEXT_START.exec(text);
  }
}

function cssEscape(token) {
  return token.replace(/[!"#$%&'()*+,.\/:;<=>?@[\\\]^`{|}~]/g, (c) => `\\${c}`);
}

// A class can be a plain, hand-authored CSS rule in tailwind.css rather than
// a Tailwind-generated utility (e.g. .scrollbar-hide). build() only grows for
// candidate-driven output, so those never show growth; fall back to checking
// whether the selector is already present in the compiled text verbatim.
let prevLen = result.build([]).length;
const failures = [];
const banned = [];
for (const [token, byFile] of found) {
  const bannedReason = bannedReasonByToken.get(token);
  if (bannedReason) {
    banned.push({ token, reason: bannedReason, byFile });
  }
  if (allowlist.has(token)) {
    continue;
  }
  const out = result.build([token]);
  const grew = out.length > prevLen;
  prevLen = out.length;
  const scanned = scannedCandidates.has(token);
  const staticMatch = !grew && out.includes(`.${cssEscape(token)}`);
  if ((!grew && !staticMatch) || !scanned) {
    failures.push({ token, grew, scanned, byFile });
  }
}

// These tokens must never compile: they sit outside --background-color-*/
// --border-color-*, so growth means the corresponding --color-* registration
// leaked into theme.css. Probed directly against build(), independent of
// whether the scanner ever sees the string in source.
const leaked = [];
for (const { token, namespace } of NAMESPACE_GUARD_CLASSES) {
  const out = result.build([token]);
  const grew = out.length > prevLen;
  prevLen = out.length;
  if (grew) {
    leaked.push({ token, namespace });
  }
}

if (failures.length === 0 && banned.length === 0 && leaked.length === 0) {
  console.info(
    `checked ${found.size} class-like tokens across ${files.length} files, all compile.`,
  );
  process.exit(0);
}

if (leaked.length > 0) {
  console.error(`${leaked.length} namespace-guard token(s) now compile:\n`);
  for (const { token, namespace } of leaked) {
    console.error(
      `  ${token}  (now emits CSS; ${namespace} must stay out of --color-*, see web/internal/ui/theme.css header)`,
    );
  }
  console.error("");
}

if (banned.length > 0) {
  banned.sort((a, b) => a.token.localeCompare(b.token));
  console.error(`${banned.length} banned class-like token(s) found:\n`);
  for (const { token, reason, byFile } of banned) {
    console.error(`  ${token}  (${reason})`);
    for (const [file, lines] of byFile) {
      console.error(`    ${file}:${[...lines].sort((a, b) => a - b).join(",")}`);
    }
  }
  console.error("");
}

if (failures.length > 0) {
  failures.sort((a, b) => a.token.localeCompare(b.token));
  console.error(`${failures.length} class-like token(s) generate no CSS:\n`);
  for (const { token, scanned, byFile } of failures) {
    const reason = scanned
      ? "no matching utility or theme value"
      : "not found by Tailwind's source scanner";
    console.error(`  ${token}  (${reason})`);
    for (const [file, lines] of byFile) {
      console.error(`    ${file}:${[...lines].sort((a, b) => a - b).join(",")}`);
    }
  }
  console.error(
    `\nIf this is a legitimate dynamically-constructed class, add it to ${path.relative(dashboardDir, allowlistPath)}.`,
  );
}
process.exit(1);
