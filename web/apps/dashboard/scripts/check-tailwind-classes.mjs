#!/usr/bin/env node
// @tailwindcss/node and @tailwindcss/oxide are resolved through
// @tailwindcss/postcss so this compiles with the exact version `next build`
// ships, even when the `tailwindcss` catalog pin disagrees.

import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath } from "node:url";

const dashboardDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const require = createRequire(path.join(dashboardDir, "package.json"));
const cssEntry = path.join(dashboardDir, "styles/tailwind.css");
const allowlistPath = path.join(dashboardDir, "scripts/tailwind-class-allowlist.json");

const postcssRequire = createRequire(require.resolve("@tailwindcss/postcss"));
const { compile } = postcssRequire("@tailwindcss/node");
const { Scanner } = postcssRequire("@tailwindcss/oxide");

const allowlist = new Set(JSON.parse(fs.readFileSync(allowlistPath, "utf8")));

const BANNED_CLASSES = [
  {
    token: "border-border",
    reason:
      "redundant: the base rule in web/internal/ui/theme.css " +
      "already sets border-color. Write bare `border` instead.",
  },
];
const bannedReasonByToken = new Map(BANNED_CLASSES.map(({ token, reason }) => [token, reason]));

const BANNED_PATTERNS = [
  {
    pattern: /^rounded(?:-[a-z]{1,2})?-[[(]/,
    reason:
      "arbitrary radius: use the Tailwind scale (xs 2px, sm 4px, md 6px, lg 8px, xl 12px, 2xl 16px, 3xl 24px)",
  },
];

function bannedReasonFor(token) {
  const exact = bannedReasonByToken.get(token);
  if (exact) {
    return exact;
  }
  const utility = splitTopLevel(token, ":").at(-1);
  for (const { pattern, reason } of BANNED_PATTERNS) {
    if (pattern.test(utility)) {
      return reason;
    }
  }
  return undefined;
}

const ROLE_NAMES = ["raised", "table-header", "input", "strong", "hairline"];

const css = fs.readFileSync(cssEntry, "utf8");
const base = path.dirname(cssEntry);
const result = await compile(css, { base, onDependency: () => {} });

// Mirrors @tailwindcss/postcss: an unset root falls back to the app root, not
// the CSS file's directory.
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
    const prefix = last.slice(0, bracketIdx);
    return prefix === "" || /^-?[a-z][a-z0-9]*(-[a-z0-9]*)*-$/i.test(prefix);
  }
  return /^-?[a-z][a-z0-9]*(-[a-z0-9]+)+$/i.test(last);
}

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

// Hand-authored rules in tailwind.css (e.g. .scrollbar-hide) never grow
// build(), so a verbatim selector match stands in. A token must also be found
// by Tailwind's own scanner: one it cannot extract ships to production missing.
let prevLen = result.build([]).length;
const failures = [];
const banned = [];
for (const [token, byFile] of found) {
  const bannedReason = bannedReasonFor(token);
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

const declaredColors = new Set(
  [...result.build([]).matchAll(/(?<![\w-])--color-([a-zA-Z][\w-]*)\s*:/g)].map((m) => m[1]),
);
const leaked = ROLE_NAMES.filter((name) => declaredColors.has(name));

if (failures.length === 0 && banned.length === 0 && leaked.length === 0) {
  console.info(
    `checked ${found.size} class-like tokens across ${files.length} files, all compile.`,
  );
  process.exit(0);
}

if (leaked.length > 0) {
  console.error(`${leaked.length} role token(s) registered in the --color-* namespace:\n`);
  for (const name of leaked) {
    console.error(
      `  --color-${name}  (move it to --background-color-* or --border-color-*, see web/internal/ui/theme.css header)`,
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
