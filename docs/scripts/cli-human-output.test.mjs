import assert from "node:assert/strict";
import fs from "node:fs";
import test from "node:test";
import { migrateManifest, migratePage, readResourcePaths } from "./cli-human-output.mjs";

const resources = readResourcePaths(fs.readFileSync(new URL("../../cmd/api/resources.go", import.meta.url), "utf8"));

test("every mapped invocation migrates without changing flags or URLs", () => {
  for (const [legacy, canonical] of resources) {
    const [group, action] = legacy.split(" ");
    const url = `/compute/cli/${group}/${action}`;
    const source = `unkey api ${legacy} --output=json\n[${action}](${url})\n`;
    const actual = migratePage(source, "compute/guide.mdx", resources);
    assert.equal(actual, `unkey ${canonical} --output=json\n[${canonical}](${url})\n`);
    assert.equal(migratePage(actual, "compute/guide.mdx", resources), actual);
  }
});

test("reference pages share output guidance and keep their legacy URLs", () => {
  const source = `---
title: unkey api environments list-environment-variables
sidebarTitle: "list-environment-variables"
---

<ParamField body="--output" type="string">
  Any other value prints the request ID and data.
</ParamField>

<ParamField body="--root-key" type="string">
  Use the key stored by unkey auth login.
</ParamField>

<ParamField body="--config" type="string" default="~/.unkey/config.toml">
  Login writes a root key here.
</ParamField>
`;
  const file = "compute/cli/environments/list-environment-variables.mdx";
  const result = migratePage(source, file, resources);
  assert.match(result, /title: unkey environments variables list/);
  assert.match(result, /sidebarTitle: "variables list"/);
  assert.match(result, /import CLIOutput from "\/snippets\/cli-output.mdx"/);
  assert.match(result, /<CLIOutput \/>/);
  assert.doesNotMatch(result, /Any other value/);
  assert.match(result, /body="--output" type="enum" default="human"/);
  assert.match(result, /body="--json" type="boolean"/);
  assert.match(result, /body="--debug" type="boolean"/);
  assert.match(result, /stored credentials, including a CLI session/);
  assert.match(result, /Browser login stores a session, not a root key/);
  assert.equal(migratePage(result, file, resources), result);
});

test("manifest migration preserves request flags and non-resource commands", () => {
  const request = { name: "api-id", type: "string", required: true };
  const login = { path: "auth login", flags: [{ name: "root-key", type: "string", required: false }] };
  const source = JSON.stringify({ commands: [
    { path: "api keys create-key", flags: [request, { name: "output", type: "string", env: "UNKEY_OUTPUT", required: false }] },
    login,
  ], endpoints: [{ path: "/v2/keys.createKey" }] });
  const result = migrateManifest(source, resources);
  const manifest = JSON.parse(result);
  assert.deepEqual(manifest.commands[0].flags, [
    request,
    { name: "debug", type: "boolean", required: false },
    { name: "json", type: "boolean", required: false },
    { name: "output", type: "enum", env: "UNKEY_OUTPUT", required: false, default: "human" },
  ]);
  assert.deepEqual(manifest.commands[1], login);
  assert.deepEqual(manifest.endpoints, [{ path: "/v2/keys.createKey" }]);
  assert.equal(migrateManifest(result, resources), result);
  assert.throws(() => migrateManifest(JSON.stringify({ commands: [{ path: "api keys create-key", flags: [] }] }), resources), /Missing output flag/);
});

test("moved resources remain distinguishable in the existing navigation groups", () => {
  const result = migratePage('sidebarTitle: "create-role"\n', "api-management/cli/permissions/create-role.mdx", resources);
  assert.equal(result, 'sidebarTitle: "roles create"\n');
});

test("unknown commands and API endpoint names are not rewritten", () => {
  const source = "unkey api projects list-projects-extra\nPOST /v2/projects.listProjects\n";
  assert.equal(migratePage(source, "compute/guide.mdx", resources), source);
  assert.throws(() => readResourcePaths(""), /Cannot find/);
  assert.throws(() => readResourcePaths('var resourcePaths = map[string]string{\n broken\n}'), /Cannot parse/);
});
