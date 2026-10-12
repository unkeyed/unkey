import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { renderApi } from "./art-api.mjs";
import { renderCompute } from "./art-compute.mjs";
import { renderEditorialBuild, renderEditorialWorkspace } from "./art-editorial.mjs";
import { renderPlatform } from "./art-platform.mjs";
import { documentSvg, escape, palettes, primitives } from "./art-primitives.mjs";
import { validateArtReview } from "./art-review.mjs";

const cards = JSON.parse(fs.readFileSync(new URL("./card-art.json", import.meta.url), "utf8"));
const reviews = validateArtReview(
  JSON.parse(fs.readFileSync(new URL("./card-art-review.json", import.meta.url), "utf8")),
  [...cards.map((card) => card.id), "homepage--workspace"],
);
const approvedCards = cards.filter((card) => reviews.get(card.id).decision === "keep");

test("art generation does not reattach illustrations to content cards", () => {
  const css = fs.readFileSync(new URL("../card-illustrations.css", import.meta.url), "utf8");
  assert.doesNotMatch(css, /\.card\b|--unkey-card-image|images\/cards\/individual/);
});

test("SVG labels escape markup and cannot be compressed or undersized", () => {
  assert.equal(escape('<key name="a&b">'), "&lt;key name=&quot;a&amp;b&quot;&gt;");
  const g = primitives("light");
  assert.throws(() => g.text(40, 40, "Too small", g.p.text, 12), /too small/);
  assert.throws(() => g.rect(0, 0, -1, 40), /Invalid SVG/);
  assert.doesNotMatch(g.text(40, 40, "A readable label"), /textLength|lengthAdjust/);
});

test("every integration mark is embedded locally with its original vector geometry", () => {
  for (const theme of Object.keys(palettes)) {
    const g = primitives(theme);
    for (const name of [
      "github",
      "docker",
      "unkey",
      "typescript",
      "go",
      "python",
      "hono",
      "bun",
      "nextjs",
      "fastapi",
      "express",
      "echo",
      "cloudflare",
    ]) {
      const mark = g.brand(name, 40, 40);
      assert.match(mark, /<(path|polygon)\b/, name);
      assert.doesNotMatch(mark, /<image\b|<script\b|<metadata\b|href="https?:/, name);
    }
    assert.throws(() => g.brand("unknown", 40, 40), /Unknown brand/);
  }
});

test("every approved card has a distinct vector composition in both themes", () => {
  const illustrations = new Set();
  for (const card of approvedCards) {
    const render = card.source.startsWith("compute/")
      ? renderCompute
      : card.source.startsWith("api-management/")
        ? renderApi
        : renderPlatform;
    for (const theme of Object.keys(palettes)) {
      const content = render(card, primitives(theme));
      assert.ok(content, card.id);
      assert.ok(!illustrations.has(content), `Reused illustration: ${card.id}`);
      illustrations.add(content);
      const svg = documentSvg(card, theme, content);
      assert.match(svg, /viewBox="0 0 600 320"/);
      assert.doesNotMatch(svg, /NaN|undefined|textLength|lengthAdjust|<image\b|<script\b/, card.id);
      assert.equal((svg.match(/<title>/g) ?? []).length, 1, card.id);
    }
  }
  assert.equal(illustrations.size, approvedCards.length * 2);
});

test("quality review requires an explicit decision for every card", () => {
  const keep = { id: "example", decision: "keep", reason: "Clear purpose and restrained layout" };
  assert.equal(validateArtReview([keep], ["example"]).get("example").decision, "keep");
  assert.throws(() => validateArtReview([], ["example"]), /needs quality review/);
  assert.throws(() => validateArtReview([keep, keep], ["example"]), /Duplicate/);
  assert.throws(() => validateArtReview([keep], []), /Unknown/);
  assert.throws(
    () => validateArtReview([{ ...keep, decision: "maybe" }], ["example"]),
    /Invalid decision/,
  );
  assert.throws(() => validateArtReview([{ ...keep, reason: "" }], ["example"]), /Missing reason/);
});

test("rejected SVGs are absent and approved artwork remains on disk", () => {
  for (const card of cards) {
    for (const theme of Object.keys(palettes)) {
      const file = new URL(`../images/cards/individual/${card.id}-${theme}.svg`, import.meta.url);
      assert.equal(fs.existsSync(file), reviews.get(card.id).decision === "keep", card.id);
    }
  }
});

test("Compute overview illustrations retain the context of their destination", () => {
  const expectedLabels = new Map([
    [4, ["Project", "App", "Production", "Preview"]],
    [5, ["Pending", "Building", "Deploying", "Ready"]],
    [13, ["Build context", "/services/api", "Dockerfile"]],
    [14, ["CPU", "0.25", "vCPU", "Memory", "256", "MiB", "8080"]],
    [18, ["Monthly budget", "50%", "75%", "100%"]],
    [19, ["TypeScript", "Go", "Python", "Unkey API"]],
  ]);
  for (const [index, labels] of expectedLabels) {
    const card = cards.find(({ id }) => id === `compute--index--${index}`);
    assert.ok(card);
    const content = renderCompute(card, primitives("light"));
    for (const label of labels) assert.ok(content.includes(`>${label}</text>`), label);
  }
  assert.throws(
    () =>
      renderCompute(
        { source: "compute/index.mdx", id: "compute--index--unknown" },
        primitives("light"),
      ),
    /Missing Compute overview composition/,
  );
});

test("editorial overview artwork uses flat surfaces and monospace technical values", () => {
  for (const theme of Object.keys(palettes)) {
    const g = primitives(theme);
    const compositions = [
      renderEditorialBuild(g),
      renderEditorialWorkspace(g),
      ...cards
        .filter((card) => card.source === "api-management/index.mdx")
        .map((card) => renderApi(card, g)),
    ];
    for (const content of compositions) {
      assert.doesNotMatch(content, /filter=|url\(#surface\)|<rect\b[^>]*fill="#202525"/);
      assert.match(content, /font-family="ui-monospace/);
      for (const radius of content.matchAll(/\brx="([\d.]+)"/g)) assert.ok(Number(radius[1]) <= 8);
    }
  }
});

test("API concepts distinguish customer keys from the root credential", () => {
  const card = cards.find(({ id }) => id === "api-management--index--2");
  assert.ok(card);
  const content = renderApi(card, primitives("light"));
  for (const value of ["Keyspace", "Key A", "Key B", "Identity", "Root key", "Unkey API"]) {
    assert.ok(content.includes(`>${value}</text>`), value);
  }
});
