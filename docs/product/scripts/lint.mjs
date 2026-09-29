#!/usr/bin/env node
// Repo-specific lint for the rebuilt product docs. Mintlify's own checks
// (`mise run docs-check`) validate structure and links; this script enforces
// the writing and coverage rules from the docs rebuild plan that those checks
// cannot know about: traceable sources, unique titles, generated coverage of
// CLI commands, error codes, and API endpoints, and a few house-style rules.
//
// Usage, from the repo root:
//
//   node docs/product/scripts/lint.mjs [docsDir]
//
// Every violation is printed as `path:line: message` and the exit code is 1
// when any violation exists. Warnings never change the exit code. The script
// has no dependencies so it runs with the Node pinned in mise alone.

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

// Directories that may hold CLI command reference pages. A command's page is
// found by matching its path segments against the page path under one of them.
export const CLI_PAGE_DIRS = ["compute/cli", "api-management/cli", "platform/cli"];

// The product CLI tabs hold nothing but command pages, so every page there must
// name a command or group. Platform's CLI area also carries prose (install,
// authentication, shared flags) and is exempt from that reverse check.
export const CLI_REFERENCE_ONLY_DIRS = ["compute/cli", "api-management/cli"];

// A fenced block whose first command is one of these must be tagged `bash` so
// every shell sample on the site highlights and copies the same way. The check
// only looks at untagged blocks and blocks tagged with another shell-ish name;
// `export` or `go` at the start of a TypeScript or Go sample is not a command.
const SHELL_COMMANDS = new Set([
  "curl", "npm", "npx", "go", "mise", "unkey", "git", "pnpm", "docker", "mint", "node", "export", "cd",
]);
const SHELL_LIKE_TAGS = new Set(["", "sh", "shell", "zsh", "console", "shell-session", "text", "txt", "plaintext", "plain"]);

const ENDPOINT_REF = /^(GET|POST|PUT|PATCH|DELETE) (\/v2\/\S+)$/;
// The product whose API reference tab lists every endpoint from every product.
const PLATFORM_PRODUCT = "platform";
const HOW_TO_FIX_HEADING = /^##+ .*how to fix/i;

function toPosix(p) {
  return p.split(path.sep).join("/");
}

function walkMdx(dir) {
  const out = [];
  const visit = (current) => {
    for (const entry of fs.readdirSync(current, { withFileTypes: true })) {
      if (entry.name.startsWith(".") || entry.name === "node_modules") {
        continue;
      }
      const abs = path.join(current, entry.name);
      if (entry.isDirectory()) {
        visit(abs);
      } else if (entry.isFile() && entry.name.endsWith(".mdx")) {
        out.push(abs);
      }
    }
  };
  visit(dir);
  return out.sort();
}

function unquote(value) {
  const v = value.trim();
  if ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("'") && v.endsWith("'"))) {
    return v.slice(1, -1);
  }
  return v;
}

// Parses the YAML subset frontmatter uses here: scalar keys, block lists, and
// inline `[a, b]` lists. Nested maps are recorded as empty so a missing value
// is still detected. Returns null when the file has no frontmatter block.
export function parseFrontmatter(lines) {
  if (lines[0] !== "---") {
    return null;
  }
  const end = lines.indexOf("---", 1);
  if (end === -1) {
    return null;
  }
  const data = {};
  const keyLines = {};
  let listKey = null;
  for (let i = 1; i < end; i++) {
    const line = lines[i];
    const item = line.match(/^\s+-\s*(.*)$/);
    if (item && listKey !== null) {
      data[listKey].push(unquote(item[1]));
      continue;
    }
    const kv = line.match(/^([A-Za-z0-9_-]+):\s*(.*)$/);
    if (!kv) {
      continue;
    }
    const [, key, rest] = kv;
    keyLines[key] = i + 1;
    if (rest === "") {
      data[key] = [];
      listKey = key;
    } else if (rest.startsWith("[") && rest.endsWith("]")) {
      const inner = rest.slice(1, -1).trim();
      data[key] = inner === "" ? [] : inner.split(",").map(unquote);
      listKey = null;
    } else {
      data[key] = unquote(rest);
      listKey = null;
    }
  }
  return { data, keyLines, endLine: end + 1 };
}

function productSlug(name) {
  return name.trim().toLowerCase().replace(/\s+/g, "-");
}

// Walks docs.json navigation and returns every page reference with whether it
// sits inside a hidden group, plus every endpoint listing keyed by product.
function collectNavigation(docsJson) {
  const pages = new Map();
  const endpoints = [];
  const visit = (node, hidden, product) => {
    if (Array.isArray(node)) {
      for (const child of node) {
        visit(child, hidden, product);
      }
      return;
    }
    if (node === null || typeof node !== "object") {
      return;
    }
    const nowHidden = hidden || node.hidden === true;
    // A product entry's `product` is its display name ("API management"); lint keys
    // products by slug ("api-management") to match tag_to_product in the manifest.
    const nowProduct = typeof node.product === "string" ? productSlug(node.product) : product;
    for (const [key, value] of Object.entries(node)) {
      if (key === "pages" && Array.isArray(value)) {
        for (const entry of value) {
          if (typeof entry !== "string") {
            visit(entry, nowHidden, nowProduct);
            continue;
          }
          if (ENDPOINT_REF.test(entry)) {
            endpoints.push({ ref: entry, product: nowProduct ?? null });
          } else if (!pages.has(entry) || (pages.get(entry) && !nowHidden)) {
            pages.set(entry, nowHidden);
          }
        }
      } else if (key === "root" && typeof value === "string") {
        if (!pages.has(value) || (pages.get(value) && !nowHidden)) {
          pages.set(value, nowHidden);
        }
      } else {
        visit(value, nowHidden, nowProduct);
      }
    }
  };
  visit(docsJson.navigation ?? {}, false, undefined);
  return { pages, endpoints };
}

// Counts sentences as terminators followed by whitespace and a capital or digit,
// after removing the abbreviations a description is likely to contain, so
// "e.g." or "v1.0 SDK" does not read as a sentence break.
function countSentences(text) {
  const stripped = text.replace(/\b(e\.g\.|i\.e\.|etc\.|vs\.|cf\.|approx\.|no\.)/gi, "");
  const breaks = stripped.match(/[.!?]+\s+(?=[A-Z0-9])/g) ?? [];
  return breaks.length + 1;
}

function lineOf(text, needle) {
  const idx = text.indexOf(needle);
  if (idx === -1) {
    return 1;
  }
  return text.slice(0, idx).split("\n").length;
}

function parseParamFields(text) {
  const fields = [];
  const tagRe = /<ParamField\b([^>]*)>/g;
  let m;
  while ((m = tagRe.exec(text)) !== null) {
    const attrs = {};
    const attrRe = /([A-Za-z:-]+)(?:=(?:"([^"]*)"|'([^']*)'|\{([^}]*)\}))?/g;
    let a;
    while ((a = attrRe.exec(m[1])) !== null) {
      const raw = a[2] ?? a[3] ?? (a[4] !== undefined ? unquote(a[4]) : undefined);
      attrs[a[1]] = raw === undefined ? true : raw;
    }
    if (typeof attrs.body === "string" && attrs.body.startsWith("--")) {
      fields.push({ flag: attrs.body.slice(2), attrs, line: text.slice(0, m.index).split("\n").length });
    }
  }
  return fields;
}

function suffixMatches(items, segments) {
  return items.filter((item) => {
    const parts = item.path.split(" ");
    if (parts.length < segments.length) {
      return false;
    }
    return segments.every((seg, i) => parts[parts.length - segments.length + i] === seg);
  });
}

export function lint({ docsDir, repoRoot, manifestPath, constantsPath, pendingPath }) {
  docsDir = path.resolve(docsDir);
  repoRoot = path.resolve(repoRoot);
  manifestPath ??= path.join(docsDir, "scripts", "manifest.json");
  constantsPath ??= path.join(repoRoot, "pkg", "codes", "constants_gen.go");
  pendingPath ??= path.join(docsDir, "scripts", "pending-upstream.txt");

  const violations = [];
  const warnings = [];
  const notes = [];
  const rel = (abs) => toPosix(path.relative(repoRoot, abs));
  const report = (abs, line, message) => violations.push({ file: rel(abs), line, message });
  const warn = (abs, line, message) => warnings.push({ file: rel(abs), line, message });

  const docsJsonPath = path.join(docsDir, "docs.json");
  let docsJson = null;
  let docsJsonText = "";
  if (!fs.existsSync(docsJsonPath)) {
    report(docsJsonPath, 1, "docs.json not found");
  } else {
    docsJsonText = fs.readFileSync(docsJsonPath, "utf8");
    try {
      docsJson = JSON.parse(docsJsonText);
    } catch (err) {
      report(docsJsonPath, 1, `docs.json is not valid JSON: ${err.message}`);
    }
  }
  const nav = docsJson ? collectNavigation(docsJson) : { pages: new Map(), endpoints: [] };

  const pages = walkMdx(docsDir).map((abs) => {
    const text = fs.readFileSync(abs, "utf8");
    const lines = text.split("\n");
    const relDocs = toPosix(path.relative(docsDir, abs));
    const segments = relDocs.split("/");
    const fm = parseFrontmatter(lines);
    return {
      abs,
      text,
      lines,
      relDocs,
      ref: relDocs.replace(/\.mdx$/, ""),
      segments,
      fm,
      isSnippet: segments[0] === "snippets",
      isApiRef: segments.includes("api-reference"),
      isChangelog: segments.includes("changelog"),
      isSdk: segments.includes("sdks"),
      hiddenByFrontmatter: fm !== null && (fm.data.hidden === true || fm.data.hidden === "true"),
    };
  });

  // House style that applies to every file, snippets and docs.json included.
  const styleTargets = pages.map((p) => ({ abs: p.abs, lines: p.lines }));
  if (docsJsonText) {
    styleTargets.push({ abs: docsJsonPath, lines: docsJsonText.split("\n") });
  }
  for (const target of styleTargets) {
    // The docs-link rule is about prose links, which must be root-relative. A
    // fenced block can legitimately contain the absolute URL, because that is
    // the literal value the API puts in an error's `type` field, so fences are
    // exempt from it. Em dashes are prose either way.
    let fence = null;
    target.lines.forEach((line, i) => {
      if (line.includes("\u2014")) {
        report(target.abs, i + 1, "em dash (U+2014); use a comma, colon, or separate sentences");
      }
      const open = line.match(/^\s*(`{3,})/);
      if (fence === null && open) {
        fence = open[1].length;
        return;
      }
      if (fence !== null) {
        if (open && open[1].length >= fence && line.trim() === "`".repeat(open[1].length)) {
          fence = null;
        }
        return;
      }
      if (/https?:\/\/(www\.)?unkey\.com\/docs/.test(line)) {
        report(target.abs, i + 1, "absolute https://unkey.com/docs link; use a root-relative path");
      }
    });
  }

  for (const page of pages) {
    page.lines.forEach((line, i) => {
      if (/<CardGroup\b/.test(line)) {
        report(page.abs, i + 1, "<CardGroup> is not used on this site; use <Columns>");
      }
    });
    checkFences(page, report);
    if (!page.isSnippet) {
      checkImages(page, report);
    }
  }

  // Frontmatter rules. Snippets are fragments without frontmatter, and
  // API reference overrides inherit title and description from the spec.
  const titles = new Map();
  for (const page of pages) {
    if (page.isSnippet) {
      continue;
    }
    if (page.fm === null) {
      report(page.abs, 1, "missing frontmatter block");
      continue;
    }
    const { data, keyLines } = page.fm;
    const at = (key) => keyLines[key] ?? 1;
    const inheritsFromSpec = page.isApiRef && typeof data.openapi === "string";
    if (!inheritsFromSpec) {
      if (typeof data.title !== "string" || data.title.trim() === "") {
        report(page.abs, at("title"), "frontmatter title is missing or empty");
      } else {
        const list = titles.get(data.title) ?? [];
        list.push(page);
        titles.set(data.title, list);
      }
      if (typeof data.description !== "string" || data.description.trim() === "") {
        report(page.abs, at("description"), "frontmatter description is missing or empty");
      } else {
        // Descriptions render under the title and feed llms.txt, so they must read as one plain
        // sentence a person would write, not a product-prefixed list of everything on the page.
        const desc = data.description.trim();
        if (/^(Compute|API Management|Platform)\s*:/.test(desc)) {
          report(page.abs, at("description"), "frontmatter description must not start with a product prefix");
        }
        if (desc.length > 120) {
          report(page.abs, at("description"), `frontmatter description is ${desc.length} characters; keep it to one short sentence (max 120)`);
        }
        if (countSentences(desc) > 1) {
          report(page.abs, at("description"), "frontmatter description must be a single sentence");
        }
      }
    }

    const needsSources = !page.isApiRef && !page.isChangelog;
    const sources = data.sources;
    if (needsSources && (!Array.isArray(sources) || sources.length === 0)) {
      report(page.abs, at("sources"), "frontmatter sources must list at least one repo file or https:// URL the page was written from");
    }
    if (Array.isArray(sources)) {
      let httpsCount = 0;
      for (const entry of sources) {
        const entryLine = findSourceLine(page, entry) ?? at("sources");
        if (/^https:\/\//.test(entry)) {
          httpsCount++;
          continue;
        }
        if (/^[a-z]+:\/\//.test(entry)) {
          report(page.abs, entryLine, `source "${entry}" must use https://`);
          continue;
        }
        const abs = path.resolve(repoRoot, entry);
        if (!fs.existsSync(abs)) {
          report(page.abs, entryLine, `source "${entry}" does not exist in the repo`);
        } else if (fs.statSync(abs).isDirectory()) {
          report(page.abs, entryLine, `source "${entry}" is a directory; cite the specific file`);
        }
      }
      if (page.isSdk && httpsCount === 0) {
        report(page.abs, at("sources"), "SDK pages must cite at least one https:// source (the SDK repository or its published docs)");
      }
    }

    const inNav = nav.pages.has(page.ref);
    if (!page.isApiRef && !inNav && !page.hiddenByFrontmatter) {
      report(page.abs, 1, `page "${page.ref}" is not referenced in docs.json and does not set hidden: true`);
    }
  }

  for (const [title, list] of titles) {
    if (list.length < 2) {
      continue;
    }
    for (const page of list) {
      const others = list.filter((p) => p !== page).map((p) => rel(p.abs)).join(", ");
      report(page.abs, page.fm.keyLines.title ?? 1, `title "${title}" is also used by ${others}; titles must be unique across the site`);
    }
  }

  // Hidden pages may link to each other freely; visible pages must not lead a
  // reader into content that is deliberately not navigable yet.
  const hiddenRefs = new Set();
  for (const [ref, hidden] of nav.pages) {
    if (hidden) {
      hiddenRefs.add(ref);
    }
  }
  for (const page of pages) {
    if (page.hiddenByFrontmatter) {
      hiddenRefs.add(page.ref);
    }
  }
  for (const page of pages) {
    if (page.isSnippet || hiddenRefs.has(page.ref)) {
      continue;
    }
    page.lines.forEach((line, i) => {
      const linkRe = /(?:\]\(|href=["'])(\/[^)"'\s#?]*)/g;
      let m;
      while ((m = linkRe.exec(line)) !== null) {
        const target = m[1].replace(/\.mdx$/, "").replace(/\/$/, "").replace(/^\//, "") || "index";
        if (hiddenRefs.has(target) || hiddenRefs.has(`${target}/index`)) {
          report(page.abs, i + 1, `links to hidden page /${target}`);
        }
      }
    });
  }

  // Coverage rules driven by generated inputs.
  if (!fs.existsSync(constantsPath)) {
    notes.push(`skipping error coverage: ${rel(constantsPath)} not found (run mise run generate)`);
  } else {
    checkErrorCoverage({ constantsPath, docsDir, pages, report, rel });
  }

  if (!fs.existsSync(manifestPath)) {
    notes.push(`skipping CLI and endpoint coverage: ${rel(manifestPath)} not found (run mise run generate)`);
  } else {
    let manifest = null;
    const manifestText = fs.readFileSync(manifestPath, "utf8");
    try {
      manifest = JSON.parse(manifestText);
    } catch (err) {
      report(manifestPath, 1, `manifest is not valid JSON: ${err.message}`);
    }
    if (manifest) {
      checkCliCoverage({ manifest, manifestPath, manifestText, pages, report, rel });
      if (docsJson) {
        const pending = readPending(pendingPath);
        checkEndpointCoverage({ manifest, manifestText, manifestPath, nav, docsJsonPath, docsJsonText, pending, report, warn, rel });
      }
    }
  }

  const order = (a, b) => (a.file === b.file ? a.line - b.line : a.file < b.file ? -1 : 1);
  violations.sort(order);
  warnings.sort(order);
  return { violations, warnings, notes };
}

function findSourceLine(page, entry) {
  const escaped = entry.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const item = new RegExp(`^\\s*-\\s*["']?${escaped}["']?\\s*,?\\s*$`);
  for (let i = 0; i < page.fm.endLine; i++) {
    if (item.test(page.lines[i])) {
      return i + 1;
    }
  }
  return null;
}

// Tracks fences so that `![` inside a code sample is not mistaken for an
// image, and Frame nesting so an image on the same line as its Frame passes.
function checkImages(page, report) {
  let frameDepth = 0;
  let fence = null;
  page.lines.forEach((line, i) => {
    const open = line.match(/^\s*(`{3,})/);
    if (fence === null && open) {
      fence = open[1].length;
      return;
    }
    if (fence !== null) {
      if (open && open[1].length >= fence && line.trim() === "`".repeat(open[1].length)) {
        fence = null;
      }
      return;
    }
    const imgRe = /<img\b|!\[[^\]]*\]\(/g;
    let m;
    while ((m = imgRe.exec(line)) !== null) {
      const before = line.slice(0, m.index);
      const opensBefore = (before.match(/<Frame\b/g) ?? []).length - (before.match(/<\/Frame>/g) ?? []).length;
      if (frameDepth + opensBefore <= 0) {
        report(page.abs, i + 1, "image is not wrapped in <Frame>");
      }
    }
    frameDepth += (line.match(/<Frame\b/g) ?? []).length - (line.match(/<\/Frame>/g) ?? []).length;
  });
}

function checkFences(page, report) {
  let fence = null;
  page.lines.forEach((line, i) => {
    const m = line.match(/^\s*(`{3,})(.*)$/);
    if (fence === null) {
      if (m) {
        fence = { len: m[1].length, lang: m[2].trim().split(/\s+/)[0] ?? "", start: i + 1, body: [] };
      }
      return;
    }
    if (m && m[1].length >= fence.len && m[2].trim() === "") {
      const first = fence.body.map((l) => l.trim()).find((l) => l !== "" && !l.startsWith("#"));
      if (first) {
        const word = first.replace(/^\$\s+/, "").replace(/^sudo\s+/, "").split(/\s+/)[0];
        if (SHELL_COMMANDS.has(word) && SHELL_LIKE_TAGS.has(fence.lang)) {
          const tagged = fence.lang === "" ? "has no language tag" : `is tagged \`${fence.lang}\``;
          report(page.abs, fence.start, `code block runs \`${word}\` but ${tagged}; use \`\`\`bash`);
        }
      }
      fence = null;
      return;
    }
    fence.body.push(line);
  });
}

function checkErrorCoverage({ constantsPath, docsDir, pages, report, rel }) {
  const byRef = new Map(pages.map((p) => [p.ref, p]));
  const lines = fs.readFileSync(constantsPath, "utf8").split("\n");
  lines.forEach((line, i) => {
    const m = line.match(/URN\s*=\s*"(err:[^"]+)"/);
    if (!m) {
      return;
    }
    const parts = m[1].split(":");
    if (parts.length < 4) {
      report(constantsPath, i + 1, `URN ${m[1]} does not have the err:<system>:<category>:<specific> shape`);
      return;
    }
    const ref = `errors/${parts.slice(1).join("/")}`;
    const page = byRef.get(ref);
    if (!page) {
      report(constantsPath, i + 1, `${m[1]} has no page at ${rel(path.join(docsDir, `${ref}.mdx`))}`);
      return;
    }
    if (!page.lines.some((l) => HOW_TO_FIX_HEADING.test(l))) {
      report(page.abs, 1, `error page for ${m[1]} has no "How to fix" heading; it is still a generator stub`);
    }
  });
}

function checkCliCoverage({ manifest, manifestPath, manifestText, pages, report, rel }) {
  const leaves = manifest.commands ?? [];
  const groups = manifest.groups ?? [];
  const pagesForLeaf = new Map(leaves.map((l) => [l.path, []]));
  const dirs = CLI_PAGE_DIRS.map((d) => `${d}/`);

  for (const page of pages) {
    const dir = dirs.find((d) => page.relDocs.startsWith(d));
    if (!dir) {
      continue;
    }
    const segments = page.ref.slice(dir.length).split("/");
    if (segments.length === 1 && segments[0] === "index") {
      continue;
    }
    const isIndex = segments[segments.length - 1] === "index";
    const lookup = isIndex ? segments.slice(0, -1) : segments;
    const leafMatches = isIndex ? [] : suffixMatches(leaves, lookup);
    if (leafMatches.length > 1) {
      report(page.abs, 1, `page matches several CLI commands (${leafMatches.map((l) => l.path).join(", ")}); place it in its group's directory`);
      continue;
    }
    if (leafMatches.length === 1) {
      const leaf = leafMatches[0];
      pagesForLeaf.get(leaf.path).push(page);
      checkFlagTable(page, leaf, report);
      continue;
    }
    const referenceOnly = CLI_REFERENCE_ONLY_DIRS.some((d) => page.relDocs.startsWith(`${d}/`));
    if (referenceOnly && suffixMatches(groups, lookup).length === 0) {
      report(page.abs, 1, `page does not name a CLI command or group from ${rel(manifestPath)}`);
    }
  }

  for (const leaf of leaves) {
    const found = pagesForLeaf.get(leaf.path);
    const line = lineOf(manifestText, `"path": "${leaf.path}"`);
    if (found.length === 0) {
      report(manifestPath, line, `CLI command \`unkey ${leaf.path}\` has no page under ${CLI_PAGE_DIRS.join("/ or ")}/`);
    } else if (found.length > 1) {
      report(manifestPath, line, `CLI command \`unkey ${leaf.path}\` is documented by more than one page: ${found.map((p) => p.relDocs).join(", ")}`);
    }
  }
}

function checkFlagTable(page, leaf, report) {
  const fields = parseParamFields(page.text);
  const documented = new Map(fields.map((f) => [f.flag, f]));
  const expected = new Map((leaf.flags ?? []).map((f) => [f.name, f]));
  const cmd = `unkey ${leaf.path}`;

  for (const [name, flag] of expected) {
    const field = documented.get(name);
    if (!field) {
      const extra = [flag.type, flag.default !== undefined ? `default ${flag.default}` : null].filter(Boolean).join(", ");
      report(page.abs, 1, `missing <ParamField body="--${name}"> for \`${cmd}\` flag --${name} (${extra})`);
      continue;
    }
    if (field.attrs.type !== flag.type) {
      report(page.abs, field.line, `--${name} has type="${field.attrs.type ?? ""}" but the command declares ${flag.type}`);
    }
    if (flag.default !== undefined && field.attrs.default !== flag.default) {
      report(page.abs, field.line, `--${name} states default "${field.attrs.default ?? ""}" but the command defaults to "${flag.default}"`);
    } else if (flag.default === undefined && field.attrs.default !== undefined) {
      report(page.abs, field.line, `--${name} states default "${field.attrs.default}" but the command has no default`);
    }
  }
  for (const [name, field] of documented) {
    if (!expected.has(name)) {
      report(page.abs, field.line, `documents flag --${name}, which \`${cmd}\` does not have`);
    }
  }
}

function readPending(pendingPath) {
  if (!fs.existsSync(pendingPath)) {
    return new Set();
  }
  return new Set(
    fs
      .readFileSync(pendingPath, "utf8")
      .split("\n")
      .map((l) => l.trim())
      .filter((l) => l !== "" && !l.startsWith("#")),
  );
}

function checkEndpointCoverage({ manifest, manifestText, manifestPath, nav, docsJsonPath, docsJsonText, pending, report, warn, rel }) {
  const tagMap = manifest.tag_to_product?.tags ?? {};
  const opMap = manifest.tag_to_product?.operations ?? {};
  const endpoints = manifest.endpoints ?? [];
  const byRef = new Map(endpoints.map((e) => [`${e.method} ${e.path}`, e]));

  // Platform carries a full API reference that repeats every product's
  // endpoints on purpose, so its listings skip the one-product and
  // wrong-product checks. They must still name a real, non-deprecated endpoint,
  // and they never count as coverage: every endpoint must still be listed
  // under its own product tab.
  const platformListings = [];
  const listings = new Map();
  for (const { ref, product } of nav.endpoints) {
    const line = lineOf(docsJsonText, `"${ref}"`);
    if (!byRef.has(ref)) {
      report(docsJsonPath, line, `"${ref}" is not an endpoint in ${rel(manifestPath)}`);
      continue;
    }
    if (product === null) {
      report(docsJsonPath, line, `"${ref}" is listed outside any product`);
      continue;
    }
    if (product === PLATFORM_PRODUCT) {
      platformListings.push({ ref, line });
      continue;
    }
    const list = listings.get(ref) ?? [];
    list.push({ product, line });
    listings.set(ref, list);
  }

  for (const { ref, line } of platformListings) {
    if (byRef.get(ref).deprecated) {
      report(docsJsonPath, line, `deprecated endpoint "${ref}" is listed under ${PLATFORM_PRODUCT}; deprecated endpoints are not listed`);
    }
  }

  for (const endpoint of endpoints) {
    const ref = `${endpoint.method} ${endpoint.path}`;
    const listed = listings.get(ref) ?? [];
    const expected = opMap[endpoint.path] ?? tagMap[endpoint.tag];
    if (endpoint.deprecated) {
      for (const { product, line } of listed) {
        report(docsJsonPath, line, `deprecated endpoint "${ref}" is listed under ${product}; deprecated endpoints are not listed`);
      }
      continue;
    }
    if (expected === undefined) {
      const line = lineOf(manifestText, `"path": "${endpoint.path}"`);
      report(manifestPath, line, `endpoint ${endpoint.path} has tag "${endpoint.tag}" with no product in tag_to_product; add it to tools/docsmanifest`);
    }
    if (listed.length === 0) {
      const where = expected ? ` under ${expected}` : "";
      const message = `endpoint "${ref}" (tag ${endpoint.tag}) is not listed in any API reference tab; add it${where}`;
      if (pending.has(endpoint.path)) {
        warn(docsJsonPath, 1, `${message} (pending upstream publish)`);
      } else {
        report(docsJsonPath, 1, message);
      }
      continue;
    }
    if (listed.length > 1) {
      for (const { product, line } of listed) {
        report(docsJsonPath, line, `endpoint "${ref}" is listed under ${listed.map((l) => l.product).join(" and ")}; it belongs only under ${expected ?? "one product"}`);
      }
      continue;
    }
    const [{ product, line }] = listed;
    if (expected !== undefined && product !== expected) {
      report(docsJsonPath, line, `endpoint "${ref}" (tag ${endpoint.tag}) is listed under ${product} but belongs under ${expected}`);
    }
  }
}

function main() {
  const docsDir = process.argv[2] ?? "docs/product";
  const repoRoot = process.cwd();
  if (!fs.existsSync(docsDir)) {
    console.error(`docs-lint: ${docsDir} does not exist`);
    process.exit(2);
  }
  const { violations, warnings, notes } = lint({ docsDir, repoRoot });
  for (const note of notes) {
    console.error(`docs-lint: ${note}`);
  }
  for (const w of warnings) {
    console.log(`${w.file}:${w.line}: warning: ${w.message}`);
  }
  for (const v of violations) {
    console.log(`${v.file}:${v.line}: ${v.message}`);
  }
  console.log(`docs-lint: ${violations.length} violation${violations.length === 1 ? "" : "s"}, ${warnings.length} warning${warnings.length === 1 ? "" : "s"} in ${docsDir}`);
  process.exit(violations.length > 0 ? 1 : 0);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main();
}
