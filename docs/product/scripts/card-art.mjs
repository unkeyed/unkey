// Only the homepage ships card artwork. Content-card designs stay in the review
// manifest as remove; rejected and unused individual SVGs stay off disk.
// Generate: mise exec -- node docs/product/scripts/card-art.mjs
// Verify:   mise exec -- node docs/product/scripts/card-art.mjs --check
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { documentSvg, palettes, primitives } from "./art-primitives.mjs";
import { renderCompute } from "./art-compute.mjs";
import { renderApi } from "./art-api.mjs";
import { renderPlatform, renderWorkspaceOverview } from "./art-platform.mjs";
import { validateArtReview } from "./art-review.mjs";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const designs = JSON.parse(fs.readFileSync(path.join(root, "scripts/card-art.json"), "utf8"));
const reviews = validateArtReview(
  JSON.parse(fs.readFileSync(path.join(root, "scripts/card-art-review.json"), "utf8")),
  [...designs.map((card) => card.id), "homepage--workspace"],
);
const checkOnly = process.argv.includes("--check");

function svg(card, theme) {
  const render = card.source.startsWith("compute/")
    ? renderCompute
    : card.source.startsWith("api-management/")
      ? renderApi
      : renderPlatform;
  const content = render(card, primitives(theme));
  assert.ok(content, `Missing composition: ${card.id}`);
  return documentSvg(card, theme, content);
}

function sourceCards() {
  const found = new Map();
  for (const file of fs
    .readdirSync(root, { recursive: true })
    .filter((f) => f.endsWith(".mdx") && !f.startsWith("snippets/"))) {
    const source = fs.readFileSync(path.join(root, file), "utf8");
    for (const match of source.matchAll(/<(Card|ProductLink)\b([^>]+)>/g)) {
      const href = match[2].match(/href="([^"]+)"/)?.[1];
      const title = match[2].match(/title="([^"]+)"/)?.[1];
      assert.ok(href && title, `Unparsed card in ${file}`);
      const key = `${file}|${href}`;
      assert.ok(!found.has(key), `Ambiguous card selector ${key}`);
      found.set(key, title);
    }
  }
  return found;
}

function generate() {
  const cards = sourceCards(),
    ids = new Set(),
    signatures = new Set(),
    hashes = new Set();
  const outputs = new Map();
  const removedFiles = [];
  let css =
    "/* Generated from card-art.json and card-art-review.json.\n * Only the homepage displays illustrations. Individual content-card SVGs are not shipped.\n */\n\n";
  for (const card of designs) {
    assert.match(card.id, /^[a-z0-9_-]+$/);
    assert.ok(!ids.has(card.id), `Duplicate ID: ${card.id}`);
    ids.add(card.id);
    assert.ok(
      fs.existsSync(path.join(root, card.evidence)),
      `Missing content reference: ${card.id}`,
    );
    const key = `${card.source}|${card.href}`;
    assert.equal(cards.get(key), card.title, `Missing or changed source card: ${key}`);
    cards.delete(key);
    if (reviews.get(card.id).decision === "remove") {
      for (const theme of Object.keys(palettes))
        removedFiles.push(`images/cards/individual/${card.id}-${theme}.svg`);
      continue;
    }
    assert.ok(
      Array.isArray(card.scene.fields) &&
        card.scene.fields.length >= 2 &&
        card.scene.fields.length <= 6,
      `Invalid fields: ${card.id}`,
    );
    for (const field of card.scene.fields)
      assert.ok(
        field.length === 2 && field.every((value) => typeof value === "string" && value.length > 0),
        `Invalid field: ${card.id}`,
      );
    const signature = JSON.stringify(card.scene.fields);
    assert.ok(!signatures.has(signature), `Reused composition: ${card.id}`);
    signatures.add(signature);
    for (const theme of Object.keys(palettes)) {
      const image = svg(card, theme);
      const hash = createHash("sha256")
        .update(image.replace(/<(title|desc)>.*?<\/(title|desc)>/g, ""))
        .digest("hex");
      assert.ok(!hashes.has(hash), `Reused image: ${card.id}`);
      hashes.add(hash);
      outputs.set(`images/cards/individual/${card.id}-${theme}.svg`, image);
    }
  }
  assert.equal(cards.size, 0, `Cards without an artwork review: ${[...cards.keys()].join(", ")}`);
  if (reviews.get("homepage--workspace").decision === "remove")
    css +=
      "html .unkey-home .unkey-platform-art {\n  display: none;\n  background-image: none;\n}\n\n";
  for (const theme of Object.keys(palettes)) {
    if (reviews.get("homepage--workspace").decision === "remove") {
      removedFiles.push(`images/cards/workspace-${theme}.svg`);
      continue;
    }
    const overview = {
      title: "Unkey Platform",
      scene: {
        heading: "One workspace for both products",
        note: "Your team, Compute, and API Management share a workspace",
      },
    };
    outputs.set(
      `images/cards/workspace-${theme}.svg`,
      documentSvg(overview, theme, renderWorkspaceOverview(primitives(theme))),
    );
  }
  outputs.set("card-illustrations.css", css);
  // Targets are exact generated filenames derived from validated manifest IDs.
  // Check mode is read-only and fails if rejected artwork has been resurrected.
  for (const file of removedFiles) {
    const target = path.join(root, file);
    if (checkOnly) assert.ok(!fs.existsSync(target), `Rejected illustration still exists: ${file}`);
    else if (fs.existsSync(target)) fs.unlinkSync(target);
  }
  for (const [file, content] of outputs) {
    const target = path.join(root, file);
    if (checkOnly)
      assert.equal(fs.readFileSync(target, "utf8"), content, `Stale illustration: ${file}`);
    else {
      fs.mkdirSync(path.dirname(target), { recursive: true });
      fs.writeFileSync(target, content);
    }
  }
  console.log(
    `card-art: ${hashes.size / 2} approved card illustrations, ${removedFiles.length} rejected SVG files absent; ${checkOnly ? "verified" : "generated"}`,
  );
}

generate();
