// Fixture tests for lint.mjs. Each test builds a tiny site in a temp directory
// with its own repo root, so source paths, the error constants file, and the
// manifest are all under the test's control.
//
//   node --test docs/product/scripts/lint.test.mjs

import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { test } from "node:test";

import { lint } from "./lint.mjs";

const PRODUCTS = (computePages, apiPages) => ({
  navigation: {
    products: [
      {
        name: "Compute",
        product: "compute",
        tabs: [{ tab: "API reference", groups: [{ group: "All", pages: computePages }] }],
      },
      {
        name: "API Management",
        product: "api-management",
        tabs: [{ tab: "API reference", groups: [{ group: "All", pages: apiPages }] }],
      },
    ],
  },
});

const MANIFEST = {
  commands: [
    {
      path: "api keys create-key",
      usage: "Create a key",
      flags: [
        { name: "api-id", type: "string", required: true },
        { name: "enabled", type: "boolean", default: "true", required: false },
        { name: "body", type: "string", required: false },
      ],
    },
    { path: "deploy", usage: "Deploy", flags: [{ name: "project", type: "string", required: true }] },
  ],
  groups: [{ path: "api", usage: "" }, { path: "api keys", usage: "" }],
  endpoints: [
    { path: "/v2/keys.createKey", method: "POST", tag: "keys", deprecated: false },
    { path: "/v2/apps.createApp", method: "POST", tag: "apps", deprecated: false },
    { path: "/v2/deploy.createDeployment", method: "POST", tag: "deploy", deprecated: true },
  ],
  tag_to_product: {
    tags: { keys: "api-management", apps: "compute" },
    operations: {},
  },
};

function page(title, body = "", extra = "") {
  return `---
title: ${title}
description: "${title} description."
sources:
  - pkg/example.go
${extra}---

${body}
`;
}

// Writes a site whose docs.json lists every page under a visible group unless
// the test supplies its own navigation. Returns lint output.
function run({ files, docsJson, manifest, constants, pending, listAll = true }) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "docs-lint-"));
  const docsDir = path.join(root, "docs", "site");
  fs.mkdirSync(path.join(root, "pkg", "codes"), { recursive: true });
  fs.writeFileSync(path.join(root, "pkg", "example.go"), "package example\n");
  fs.mkdirSync(docsDir, { recursive: true });

  for (const [rel, content] of Object.entries(files)) {
    const abs = path.join(docsDir, rel);
    fs.mkdirSync(path.dirname(abs), { recursive: true });
    fs.writeFileSync(abs, content);
  }
  const pageRefs = Object.keys(files)
    .filter((f) => !f.startsWith("snippets/"))
    .map((f) => f.replace(/\.mdx$/, ""));
  const nav = docsJson ?? (listAll ? { navigation: { groups: [{ group: "All", pages: pageRefs }] } } : { navigation: {} });
  fs.writeFileSync(path.join(docsDir, "docs.json"), JSON.stringify(nav, null, 2));

  if (manifest) {
    fs.mkdirSync(path.join(docsDir, "scripts"), { recursive: true });
    fs.writeFileSync(path.join(docsDir, "scripts", "manifest.json"), JSON.stringify(manifest, null, 2));
  }
  if (pending) {
    fs.mkdirSync(path.join(docsDir, "scripts"), { recursive: true });
    fs.writeFileSync(path.join(docsDir, "scripts", "pending-upstream.txt"), pending);
  }
  if (constants) {
    fs.writeFileSync(path.join(root, "pkg", "codes", "constants_gen.go"), constants);
  }
  const result = lint({ docsDir, repoRoot: root });
  fs.rmSync(root, { recursive: true, force: true });
  return result;
}

function messages(result) {
  return result.violations.map((v) => `${v.file}:${v.line}: ${v.message}`);
}

function assertOne(result, pattern) {
  const hits = messages(result).filter((m) => pattern.test(m));
  assert.equal(hits.length, 1, `expected exactly one match for ${pattern} in:\n${messages(result).join("\n")}`);
  return hits[0];
}

function assertNone(result, pattern) {
  const hits = messages(result).filter((m) => pattern.test(m));
  assert.deepEqual(hits, [], `expected no match for ${pattern}`);
}

test("clean fixture site has no violations", () => {
  const result = run({ files: { "guide.mdx": page("Guide", "Hello.") } });
  assert.deepEqual(messages(result), []);
  assert.ok(result.notes.some((n) => n.includes("constants_gen.go")));
  assert.ok(result.notes.some((n) => n.includes("manifest.json")));
});

test("em dash is reported with its line number", () => {
  const result = run({ files: { "guide.mdx": page("Guide", "First line.\nBad — line.") } });
  const hit = assertOne(result, /em dash/);
  assert.match(hit, /^docs\/site\/guide\.mdx:9: /);
});

test("descriptions must be one short sentence without a product prefix", () => {
  const fm = (desc) => `---\ntitle: T-${desc.length}-${desc.charCodeAt(0)}\ndescription: "${desc}"\nsources:\n  - pkg/example.go\n---\n\nBody.\n`;
  const result = run({
    files: {
      "prefix.mdx": fm("Compute: deploys an image."),
      "long.mdx": fm("x".repeat(121)),
      "two.mdx": fm("Deploy an image. Then wait for it to become ready."),
      "ok.mdx": fm("Deploy a prebuilt container image to a project with one command."),
      "abbrev.mdx": fm("Filter by a status code, e.g. 429, or by a range such as 5xx."),
      "question.mdx": fm("Why did it fail? Read the build log."),
    },
  });
  assert.match(assertOne(result, /must not start with a product prefix/), /prefix\.mdx:3:/);
  assert.match(assertOne(result, /is 121 characters/), /long\.mdx/);
  const sentences = messages(result).filter((m) => /must be a single sentence/.test(m));
  assert.deepEqual(sentences.map((m) => m.split(":")[0]).sort(), ["docs/site/question.mdx", "docs/site/two.mdx"]);
  assertNone(result, /ok\.mdx/);
  assertNone(result, /abbrev\.mdx/);
});

test("missing sources is reported outside api-reference and changelog only", () => {
  const noSources = `---\ntitle: Guide\ndescription: "Desc."\n---\n\nBody.\n`;
  const result = run({
    files: {
      "guide.mdx": noSources,
      "api-reference/override.mdx": noSources.replace("Guide", "Override"),
      "changelog/index.mdx": noSources.replace("Guide", "Changelog"),
    },
  });
  const hit = assertOne(result, /sources must list/);
  assert.match(hit, /guide\.mdx/);
});

test("sources entries must be existing files or https URLs", () => {
  const result = run({
    files: {
      "guide.mdx": `---
title: Guide
description: "Desc."
sources:
  - pkg/example.go
  - https://example.com/spec
  - pkg/missing.go
  - pkg
  - http://example.com/insecure
---

Body.
`,
    },
  });
  assert.match(assertOne(result, /pkg\/missing\.go" does not exist/), /guide\.mdx:7:/);
  assert.match(assertOne(result, /"pkg" is a directory/), /guide\.mdx:8:/);
  assertOne(result, /must use https/);
  assertNone(result, /example\.go/);
  assertNone(result, /example\.com\/spec/);
});

test("images must be wrapped in Frame outside snippets", () => {
  const result = run({
    files: {
      "bare.mdx": page("Bare", "![alt](/img.png)\n\n<img src=\"/x.png\" />"),
      "framed.mdx": page("Framed", "<Frame>\n  ![alt](/img.png)\n</Frame>\n<Frame caption=\"c\"><img src=\"/x.png\" /></Frame>"),
      "fenced.mdx": page("Fenced", "```md\n![alt](/img.png)\n```"),
      "snippets/img.mdx": "![alt](/img.png)\n",
    },
  });
  const hits = messages(result).filter((m) => /not wrapped in <Frame>/.test(m));
  assert.equal(hits.length, 2);
  assert.ok(hits.every((h) => h.includes("bare.mdx")));
});

test("CardGroup is reported and Columns passes", () => {
  const result = run({
    files: {
      "a.mdx": page("A", "<CardGroup cols={2}>\n</CardGroup>"),
      "b.mdx": page("B", "<Columns cols={2}>\n</Columns>"),
    },
  });
  assert.match(assertOne(result, /CardGroup/), /a\.mdx:8:/);
});

test("shell code blocks must use bash", () => {
  const result = run({
    files: {
      "a.mdx": page("A", "```sh\ncurl https://api.unkey.com\n```\n\n```\n# comment\nnpm install unkey\n```\n\n```bash\nunkey api keys create-key\n```\n\n```json\n{\"a\": 1}\n```\n\n```typescript\nexport const GET = withUnkey(handler);\n```\n\n```go\ngo func() {}()\n```"),
    },
  });
  const hits = messages(result).filter((m) => /code block runs/.test(m));
  assert.equal(hits.length, 2);
  assert.match(hits[0], /a\.mdx:8: .*`curl`.*tagged `sh`/);
  assert.match(hits[1], /a\.mdx:12: .*`npm`.*no language tag/);
});

test("pages must be in docs.json unless hidden", () => {
  const result = run({
    files: {
      "listed.mdx": page("Listed"),
      "orphan.mdx": page("Orphan"),
      "secret.mdx": page("Secret", "", "hidden: true\n"),
      "api-reference/x.mdx": page("X"),
    },
    docsJson: { navigation: { groups: [{ group: "All", pages: ["listed"] }] } },
  });
  const hit = assertOne(result, /not referenced in docs\.json/);
  assert.match(hit, /orphan\.mdx/);
});

test("unkey.com/docs links are reported", () => {
  const result = run({ files: { "a.mdx": page("A", "See [x](https://unkey.com/docs/foo) and [y](/foo).") } });
  assertOne(result, /unkey\.com\/docs/);
});

test("duplicate titles are reported on both pages", () => {
  const result = run({ files: { "a/index.mdx": page("Overview"), "b/index.mdx": page("Overview") } });
  const hits = messages(result).filter((m) => /title "Overview" is also used/.test(m));
  assert.equal(hits.length, 2);
  assert.match(hits[0], /a\/index\.mdx:2: .*b\/index\.mdx/);
  assert.match(hits[1], /b\/index\.mdx:2: .*a\/index\.mdx/);
});

test("visible pages must not link into hidden groups", () => {
  const result = run({
    files: {
      "visible.mdx": page("Visible", "[go](/hidden/one)"),
      "hidden/one.mdx": page("One", "[peer](/hidden/two) <a href=\"/hidden/two\">x</a>"),
      "hidden/two.mdx": page("Two"),
      "fm-hidden.mdx": page("FM", "[go](/hidden/one)", "hidden: true\n"),
    },
    docsJson: {
      navigation: {
        groups: [
          { group: "Public", pages: ["visible"] },
          { group: "Secret", hidden: true, pages: ["hidden/one", "hidden/two"] },
        ],
      },
    },
  });
  const hit = assertOne(result, /links to hidden page/);
  assert.match(hit, /visible\.mdx:8: links to hidden page \/hidden\/one/);
});

test("sdks pages need an https source", () => {
  const result = run({
    files: {
      "sdks/go.mdx": page("Go SDK"),
      "sdks/ts.mdx": `---\ntitle: TS SDK\ndescription: "D."\nsources: [pkg/example.go, https://github.com/unkeyed/sdks]\n---\n\nBody.\n`,
    },
  });
  assert.match(assertOne(result, /SDK pages must cite/), /sdks\/go\.mdx/);
});

const CONSTANTS = `package codes

const (
	UserErrorsBadRequestMissing URN = "err:user:bad_request:missing_header"
	UserErrorsBadRequestStub URN = "err:user:bad_request:stub_only"
	UserErrorsBadRequestGood URN = "err:user:bad_request:good"
)
`;

test("error coverage reports missing pages and bare stubs", () => {
  const result = run({
    constants: CONSTANTS,
    files: {
      "errors/user/bad_request/stub_only.mdx": `---\ntitle: "Stub"\ndescription: "Stub."\n---\n\n<Danger>\`err:user:bad_request:stub_only\`</Danger>\n`,
      "errors/user/bad_request/good.mdx": page("Good", "<Danger>`err:user:bad_request:good`</Danger>\n\n## How to fix\n\nDo the thing."),
    },
  });
  const missing = assertOne(result, /has no page/);
  assert.match(missing, /pkg\/codes\/constants_gen\.go:4: err:user:bad_request:missing_header has no page at docs\/site\/errors\/user\/bad_request\/missing_header\.mdx/);
  const stub = assertOne(result, /generator stub/);
  assert.match(stub, /stub_only\.mdx/);
  assertNone(result, /good/);
});

function cliPage(title, fields) {
  return page(title, fields.join("\n"));
}

const KEYS_PAGE_OK = cliPage("unkey api keys create-key", [
  '<ParamField body="--api-id" type="string" required>The API.</ParamField>',
  '<ParamField body="--enabled" type="boolean" default="true">On.</ParamField>',
  '<ParamField body="--body" type="string">Raw body.</ParamField>',
]);
const DEPLOY_PAGE_OK = cliPage("unkey deploy", ['<ParamField body="--project" type="string" required>P.</ParamField>']);

test("CLI coverage passes when every leaf has one matching page", () => {
  const result = run({
    manifest: MANIFEST,
    files: {
      "api-management/cli/index.mdx": page("CLI tab"),
      "api-management/cli/keys/index.mdx": page("Keys group"),
      "api-management/cli/keys/create-key.mdx": KEYS_PAGE_OK,
      "compute/cli/deploy.mdx": DEPLOY_PAGE_OK,
    },
    docsJson: PRODUCTS(["POST /v2/apps.createApp"], ["POST /v2/keys.createKey"]),
  });
  assert.deepEqual(messages(result).filter((m) => !/not referenced in docs\.json/.test(m)), []);
});

test("prose pages under platform/cli are allowed while its command pages are still matched", () => {
  const manifest = structuredClone(MANIFEST);
  manifest.commands.push({ path: "auth login", usage: "Log in", flags: [] });
  manifest.groups.push({ path: "auth", usage: "" });
  const result = run({
    manifest,
    files: {
      "platform/cli/install.mdx": page("Install the CLI"),
      "platform/cli/auth/login.mdx": page("unkey auth login", '<ParamField body="--bogus" type="string">x</ParamField>'),
      "api-management/cli/keys/create-key.mdx": KEYS_PAGE_OK,
      "compute/cli/deploy.mdx": DEPLOY_PAGE_OK,
    },
    docsJson: PRODUCTS(["POST /v2/apps.createApp"], ["POST /v2/keys.createKey"]),
  });
  assert.deepEqual(messages(result).filter((m) => /install\.mdx/.test(m) && !/not referenced in docs\.json/.test(m)), []);
  assertOne(result, /documents flag --bogus, which `unkey auth login` does not have/);
  assertNone(result, /`unkey auth login` has no page/);
});

test("CLI page whose name is not in the manifest is reported", () => {
  const result = run({
    manifest: MANIFEST,
    files: {
      "api-management/cli/keys/create-key.mdx": KEYS_PAGE_OK,
      "api-management/cli/keys/frobnicate.mdx": page("Frob"),
      "compute/cli/deploy.mdx": DEPLOY_PAGE_OK,
    },
    docsJson: PRODUCTS(["POST /v2/apps.createApp"], ["POST /v2/keys.createKey"]),
  });
  assert.match(assertOne(result, /does not name a CLI command/), /frobnicate\.mdx/);
});

test("CLI leaf without a page is reported by name", () => {
  const result = run({
    manifest: MANIFEST,
    files: { "api-management/cli/keys/create-key.mdx": KEYS_PAGE_OK },
    docsJson: PRODUCTS(["POST /v2/apps.createApp"], ["POST /v2/keys.createKey"]),
  });
  const hits = messages(result).filter((m) => /has no page under/.test(m));
  assert.equal(hits.length, 1);
  assert.match(hits[0], /manifest\.json:\d+: CLI command `unkey deploy` has no page under compute\/cli\/ or api-management\/cli\//);
});

test("CLI leaf documented twice is reported", () => {
  const result = run({
    manifest: MANIFEST,
    files: {
      "api-management/cli/keys/create-key.mdx": KEYS_PAGE_OK,
      "compute/cli/deploy.mdx": DEPLOY_PAGE_OK,
      "api-management/cli/deploy.mdx": DEPLOY_PAGE_OK.replace("unkey deploy", "Deploy again"),
    },
    docsJson: PRODUCTS(["POST /v2/apps.createApp"], ["POST /v2/keys.createKey"]),
  });
  assertOne(result, /`unkey deploy` is documented by more than one page/);
});

test("CLI flag table drift is reported naming the flag", () => {
  const result = run({
    manifest: MANIFEST,
    files: {
      "api-management/cli/keys/create-key.mdx": cliPage("unkey api keys create-key", [
        '<ParamField body="--api-id" type="string" required>The API.</ParamField>',
        '<ParamField body="--enabled" type="boolean" default="false">On.</ParamField>',
        '<ParamField body="--extra" type="string">Not real.</ParamField>',
      ]),
      "compute/cli/deploy.mdx": cliPage("unkey deploy", ['<ParamField body="--project" type="integer" default="x">P.</ParamField>']),
    },
    docsJson: PRODUCTS(["POST /v2/apps.createApp"], ["POST /v2/keys.createKey"]),
  });
  assertOne(result, /missing <ParamField body="--body">/);
  assert.match(assertOne(result, /--enabled states default "false" but the command defaults to "true"/), /create-key\.mdx:9:/);
  assertOne(result, /documents flag --extra, which `unkey api keys create-key` does not have/);
  assertOne(result, /--project has type="integer" but the command declares string/);
  assertOne(result, /--project states default "x" but the command has no default/);
});

test("endpoint listed in wrong product or with unmapped tag is reported", () => {
  const manifest = structuredClone(MANIFEST);
  manifest.endpoints.push({ path: "/v2/widgets.list", method: "POST", tag: "widgets", deprecated: false });
  const result = run({
    manifest,
    files: {},
    docsJson: PRODUCTS(["POST /v2/keys.createKey", "POST /v2/apps.createApp", "POST /v2/widgets.list"], []),
  });
  assertOne(result, /"POST \/v2\/keys\.createKey" \(tag keys\) is listed under compute but belongs under api-management/);
  assertOne(result, /tag "widgets" with no product in tag_to_product/);
  assertNone(result, /apps\.createApp/);
});

test("endpoint listed in both products or neither is reported; deprecated unlisted is not", () => {
  const result = run({
    manifest: MANIFEST,
    files: {},
    docsJson: PRODUCTS(["POST /v2/keys.createKey"], ["POST /v2/keys.createKey"]),
  });
  const both = messages(result).filter((m) => /keys\.createKey" is listed under compute and api-management/.test(m));
  assert.equal(both.length, 2);
  assertOne(result, /"POST \/v2\/apps\.createApp" \(tag apps\) is not listed in any API reference tab; add it under compute/);
  assertNone(result, /deploy\.createDeployment/);
});

test("deprecated endpoint that is listed is reported", () => {
  const result = run({
    manifest: MANIFEST,
    files: {},
    docsJson: PRODUCTS(["POST /v2/apps.createApp", "POST /v2/deploy.createDeployment"], ["POST /v2/keys.createKey"]),
  });
  assertOne(result, /deprecated endpoint "POST \/v2\/deploy\.createDeployment" is listed under compute/);
});

test("pending upstream endpoints are warnings, not violations", () => {
  const result = run({
    manifest: MANIFEST,
    pending: "# not yet in the remote spec\n/v2/apps.createApp\n",
    files: {},
    docsJson: PRODUCTS([], ["POST /v2/keys.createKey"]),
  });
  assertNone(result, /apps\.createApp/);
  assert.equal(result.warnings.length, 1);
  assert.match(result.warnings[0].message, /apps\.createApp.*pending upstream publish/);
});

test("docs.json endpoint entry unknown to the manifest is reported", () => {
  const result = run({
    manifest: MANIFEST,
    files: {},
    docsJson: PRODUCTS(["POST /v2/apps.createApp", "POST /v2/nope.doIt"], ["POST /v2/keys.createKey"]),
  });
  assertOne(result, /"POST \/v2\/nope\.doIt" is not an endpoint in/);
});
