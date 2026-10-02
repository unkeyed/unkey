#!/usr/bin/env node
// Coverage report for the rebuilt product docs. Two checks the lint cannot
// express because they need the plan's own inventory as input:
//
//   1. Every capability in coverage-map.json (one row per Appendix A item) names
//      pages that exist under the docs dir and are reachable from docs.json,
//      or are deliberately hidden.
//   2. No page body contains a string from forbidden-claims.json (Appendix B
//      claims that must not carry over). Frontmatter is skipped because
//      `sources:` legitimately cites dashboard routes such as `[namespaceId]`.
//
// Usage, from the repo root:
//
//   node docs/product/scripts/coverage-report.mjs [docsDir]
//
// Exit code 1 on any missing capability page or forbidden hit. No dependencies.

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { parseFrontmatter } from "./lint.mjs";

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

// Returns every page ref in docs.json navigation mapped to whether it sits in a
// hidden group. Endpoint entries ("POST /v2/...") are not pages and are dropped.
export function collectNavPages(docsJson) {
  const pages = new Map();
  const visit = (node, hidden) => {
    if (Array.isArray(node)) {
      for (const item of node) {
        visit(item, hidden);
      }
      return;
    }
    if (typeof node === "string") {
      if (!/^(GET|POST|PUT|PATCH|DELETE) /.test(node)) {
        pages.set(node, hidden || pages.get(node) === true);
      }
      return;
    }
    if (node && typeof node === "object") {
      const h = hidden || node.hidden === true;
      if (typeof node.root === "string") {
        visit(node.root, h);
      }
      for (const key of ["products", "tabs", "groups", "pages", "anchors", "dropdowns", "versions", "languages"]) {
        if (node[key] !== undefined) {
          visit(node[key], h);
        }
      }
    }
  };
  visit(docsJson.navigation ?? {}, false);
  return pages;
}

export function coverageReport({ docsDir, mapPath, forbiddenPath }) {
  const missing = [];
  const forbidden = [];

  const map = JSON.parse(fs.readFileSync(mapPath ?? path.join(docsDir, "scripts", "coverage-map.json"), "utf8"));
  const claims = JSON.parse(fs.readFileSync(forbiddenPath ?? path.join(docsDir, "scripts", "forbidden-claims.json"), "utf8"));
  const docsJson = JSON.parse(fs.readFileSync(path.join(docsDir, "docs.json"), "utf8"));
  const nav = collectNavPages(docsJson);

  const seenIds = new Set();
  const coveredPages = new Set();
  for (const cap of map.capabilities) {
    if (seenIds.has(cap.id)) {
      missing.push({ id: cap.id, page: null, reason: "duplicate capability id" });
    }
    seenIds.add(cap.id);
    if (!Array.isArray(cap.pages) || cap.pages.length === 0) {
      missing.push({ id: cap.id, page: null, reason: "no pages listed" });
      continue;
    }
    for (const ref of cap.pages) {
      coveredPages.add(ref);
      const abs = path.join(docsDir, `${ref}.mdx`);
      if (!fs.existsSync(abs)) {
        missing.push({ id: cap.id, page: ref, reason: "page does not exist" });
        continue;
      }
      if (nav.has(ref)) {
        continue;
      }
      const fm = parseFrontmatter(fs.readFileSync(abs, "utf8").split("\n"));
      if (fm?.data?.hidden === "true") {
        continue;
      }
      missing.push({ id: cap.id, page: ref, reason: "page is not in docs.json and is not hidden" });
    }
  }

  const files = walkMdx(docsDir);
  for (const abs of files) {
    const lines = fs.readFileSync(abs, "utf8").split("\n");
    const fm = parseFrontmatter(lines);
    const start = fm ? fm.endLine : 0;
    for (let i = start; i < lines.length; i++) {
      for (const claim of claims.forbidden) {
        if (lines[i].includes(claim.pattern)) {
          forbidden.push({ file: toPosix(path.relative(process.cwd(), abs)), line: i + 1, pattern: claim.pattern, reason: claim.reason });
        }
      }
    }
  }

  const byProduct = {};
  for (const cap of map.capabilities) {
    byProduct[cap.product] = (byProduct[cap.product] ?? 0) + 1;
  }

  return {
    missing,
    forbidden,
    stats: {
      capabilities: map.capabilities.length,
      byProduct,
      pagesReferenced: coveredPages.size,
      pagesScanned: files.length,
      patterns: claims.forbidden.length,
    },
  };
}

function main() {
  const docsDir = process.argv[2] ?? "docs/product";
  if (!fs.existsSync(docsDir)) {
    console.error(`docs-coverage: ${docsDir} does not exist`);
    process.exit(2);
  }
  const { missing, forbidden, stats } = coverageReport({ docsDir });
  for (const m of missing) {
    console.log(`missing: ${m.id}${m.page ? ` -> ${m.page}` : ""}: ${m.reason}`);
  }
  for (const f of forbidden) {
    console.log(`${f.file}:${f.line}: forbidden "${f.pattern}" (${f.reason})`);
  }
  const products = Object.entries(stats.byProduct).map(([k, v]) => `${k} ${v}`).join(", ");
  console.log(`docs-coverage: ${stats.capabilities} capabilities (${products}) over ${stats.pagesReferenced} pages; ${missing.length} missing`);
  console.log(`docs-coverage: ${stats.patterns} forbidden patterns scanned across ${stats.pagesScanned} files; ${forbidden.length} hits`);
  process.exit(missing.length > 0 || forbidden.length > 0 ? 1 : 0);
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main();
}
