// Fixture tests for coverage-report.mjs. Each test builds a tiny site in a temp
// directory with its own coverage map and forbidden list.
//
//   node --test docs/product/scripts/coverage-report.test.mjs

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { test } from "node:test";

import { coverageReport } from "./coverage-report.mjs";

function page(title, body = "", extraFrontmatter = "") {
  return `---
title: ${title}
description: "${title} description."
sources:
  - web/apps/dashboard/app/[namespaceId]/page.tsx
${extraFrontmatter}---

${body}
`;
}

function site({ pages, nav, map, forbidden }) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "coverage-"));
  fs.mkdirSync(path.join(dir, "scripts"));
  for (const [ref, content] of Object.entries(pages)) {
    const abs = path.join(dir, `${ref}.mdx`);
    fs.mkdirSync(path.dirname(abs), { recursive: true });
    fs.writeFileSync(abs, content);
  }
  fs.writeFileSync(path.join(dir, "docs.json"), JSON.stringify({ navigation: nav }));
  fs.writeFileSync(path.join(dir, "scripts", "coverage-map.json"), JSON.stringify({ capabilities: map }));
  fs.writeFileSync(
    path.join(dir, "scripts", "forbidden-claims.json"),
    JSON.stringify({ forbidden: forbidden ?? [{ pattern: "namespaceId", reason: "stale field" }] }),
  );
  return dir;
}

const NAV = (pages, hidden = []) => ({
  products: [
    {
      name: "Compute",
      product: "compute",
      groups: [
        { group: "Guides", pages },
        { group: "Hidden", hidden: true, pages: hidden },
      ],
    },
  ],
});

test("every capability page present and in nav passes", () => {
  const dir = site({
    pages: { "compute/a": page("A", "Body about apps."), "compute/b": page("B") },
    nav: NAV(["compute/a", "compute/b"]),
    map: [{ id: "compute.ab", product: "compute", capability: "a and b", pages: ["compute/a", "compute/b"] }],
  });
  const r = coverageReport({ docsDir: dir });
  assert.deepEqual(r.missing, []);
  assert.deepEqual(r.forbidden, []);
  assert.equal(r.stats.capabilities, 1);
});

test("a missing page fails naming the capability and the page", () => {
  const dir = site({
    pages: { "compute/a": page("A") },
    nav: NAV(["compute/a"]),
    map: [{ id: "compute.gone", product: "compute", capability: "gone", pages: ["compute/a", "compute/nope"] }],
  });
  const r = coverageReport({ docsDir: dir });
  assert.equal(r.missing.length, 1);
  assert.equal(r.missing[0].id, "compute.gone");
  assert.equal(r.missing[0].page, "compute/nope");
  assert.match(r.missing[0].reason, /does not exist/);
});

test("a page that exists but is not in nav fails unless hidden", () => {
  const dir = site({
    pages: {
      "compute/a": page("A"),
      "compute/orphan": page("Orphan"),
      "compute/secret": page("Secret", "", "hidden: true\n"),
      "compute/grouped": page("Grouped"),
    },
    nav: NAV(["compute/a"], ["compute/grouped"]),
    map: [
      { id: "compute.orphan", product: "compute", capability: "orphan", pages: ["compute/orphan"] },
      { id: "compute.secret", product: "compute", capability: "secret", pages: ["compute/secret"] },
      { id: "compute.grouped", product: "compute", capability: "grouped", pages: ["compute/grouped"] },
    ],
  });
  const r = coverageReport({ docsDir: dir });
  assert.deepEqual(
    r.missing.map((m) => m.id),
    ["compute.orphan"],
  );
  assert.match(r.missing[0].reason, /not in docs.json/);
});

test("a forbidden string in a body fails naming the file and line", () => {
  const dir = site({
    pages: { "compute/a": page("A", "First line.\nUse the namespaceId field here.") },
    nav: NAV(["compute/a"]),
    map: [{ id: "compute.a", product: "compute", capability: "a", pages: ["compute/a"] }],
  });
  const r = coverageReport({ docsDir: dir });
  assert.equal(r.forbidden.length, 1);
  assert.match(r.forbidden[0].file, /compute\/a\.mdx$/);
  assert.equal(r.forbidden[0].pattern, "namespaceId");
  // Frontmatter is 6 lines plus the closing fence and a blank line; the hit is the second body line.
  const lines = fs.readFileSync(path.join(dir, "compute", "a.mdx"), "utf8").split("\n");
  assert.equal(lines[r.forbidden[0].line - 1], "Use the namespaceId field here.");
});

test("a forbidden string only in frontmatter passes", () => {
  const dir = site({
    pages: { "compute/a": page("A", "Clean body.") },
    nav: NAV(["compute/a"]),
    map: [{ id: "compute.a", product: "compute", capability: "a", pages: ["compute/a"] }],
  });
  const r = coverageReport({ docsDir: dir });
  assert.deepEqual(r.forbidden, []);
});

test("a capability with no pages and a duplicate id are reported", () => {
  const dir = site({
    pages: { "compute/a": page("A") },
    nav: NAV(["compute/a"]),
    map: [
      { id: "compute.a", product: "compute", capability: "a", pages: ["compute/a"] },
      { id: "compute.a", product: "compute", capability: "a again", pages: ["compute/a"] },
      { id: "compute.empty", product: "compute", capability: "empty", pages: [] },
    ],
  });
  const r = coverageReport({ docsDir: dir });
  assert.deepEqual(
    r.missing.map((m) => m.reason),
    ["duplicate capability id", "no pages listed"],
  );
});
