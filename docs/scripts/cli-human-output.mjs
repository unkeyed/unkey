import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

export function readResourcePaths(source) {
  const block = source.match(/var resourcePaths = map\[string\]string\{([\s\S]*?)\n\}/);
  if (!block) {
    throw new Error("Cannot find resourcePaths in cmd/api/resources.go");
  }
  const entries = [...block[1].matchAll(/^\s*"([^"]+)":\s*"([^"]+)",$/gm)];
  if (entries.length === 0 || entries.length !== block[1].trim().split("\n").length) {
    throw new Error("Cannot parse every resourcePaths entry");
  }
  return new Map(entries.map(([, legacy, canonical]) => [legacy, canonical]));
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

export function migratePage(source, relativePath, resources) {
  let text = source;
  for (const [legacy, canonical] of resources) {
    const [group, action] = legacy.split(" ");
    const invocation = new RegExp(`\\bunkey api ${escapeRegExp(legacy)}(?![\\w-])`, "g");
    text = text.replace(invocation, `unkey ${canonical}`);
    text = text.replaceAll(`\`${legacy}\``, `\`${canonical}\``);
    const link = new RegExp(`\\[(\x60?)${escapeRegExp(action)}\\1\\](\\(/(?:compute|api-management|platform)/cli/${escapeRegExp(group)}/${escapeRegExp(action)}(?:#[^)]*)?\\))`, "g");
    text = text.replace(link, `[$1${canonical}$1]$2`);
    if (relativePath.endsWith(`/cli/${group}/${action}.mdx`)) {
      const sidebar = canonical.startsWith(`${group} `) ? canonical.slice(group.length + 1) : canonical;
      text = text.replace(/^sidebarTitle:.*$/m, `sidebarTitle: "${sidebar}"`);
    }
  }

  const actions = new Map();
  for (const [legacy, canonical] of resources) {
    const action = legacy.split(" ")[1];
    actions.set(action, actions.has(action) ? null : canonical);
  }
  for (const [action, canonical] of actions) {
    if (!canonical || !action.includes("-")) {
      continue;
    }
    text = text.replace(new RegExp(`\x60${escapeRegExp(action)}(?=\x60| --)`, "g"), `\`unkey ${canonical}`);
  }

  text = text.replaceAll("Every `unkey api` command", "Every resource command");
  if (/^(?:compute|api-management)\/cli\/|^platform\/cli\/[^/]+\//.test(relativePath)) {
    const output = /<ParamField body="--output"[^>]*>\n[\s\S]*?<\/ParamField>/g;
    if (output.test(text)) {
      text = text.replace(output, '<ParamField body="--output" type="enum" default="human">\n  <CLIOutput />\n</ParamField>');
      if (!text.includes('import CLIOutput from "/snippets/cli-output.mdx";')) {
        text = text.replace(/^(---\n[\s\S]*?\n---\n)/, '$1\nimport CLIOutput from "/snippets/cli-output.mdx";\n');
      }
      for (const [name, description] of [
        ["json", "Print the full response as JSON. Takes precedence over a valid `--output` or `UNKEY_OUTPUT` value."],
        ["debug", "Print the successful response's request ID to stderr, separate from stdout."],
      ]) {
        if (!text.includes(`body="--${name}"`)) {
          text = text.replace(output, `$&\n\n<ParamField body="--${name}" type="boolean">\n  ${description}\n</ParamField>`);
        }
      }
      text = text.replace(/(<ParamField body="--root-key"[^>]*>)[\s\S]*?<\/ParamField>/g,
        '$1\n  Root key used for the request. Falls back to `UNKEY_ROOT_KEY`, then stored credentials, including a CLI session. See [CLI authentication](/platform/cli/authentication) for the full precedence order.\n</ParamField>');
      text = text.replace(/(<ParamField body="--config"[^>]*>)[\s\S]*?<\/ParamField>/g,
        '$1\n  Path of the TOML config file. Falls back to `UNKEY_CONFIG`. An explicit `--config` takes precedence over a stored CLI session. Browser login stores a session, not a root key in this file. See [CLI authentication](/platform/cli/authentication).\n</ParamField>');
    }
  }
  return text;
}

export function migrateManifest(source, resources) {
  const manifest = JSON.parse(source);
  for (const command of manifest.commands) {
    if (!command.path.startsWith("api ") || !resources.has(command.path.slice(4))) {
      continue;
    }
    const output = command.flags.find((flag) => flag.name === "output");
    if (!output) {
      throw new Error(`Missing output flag for ${command.path}`);
    }
    output.type = "enum";
    output.default = "human";
    for (const name of ["debug", "json"]) {
      if (!command.flags.some((flag) => flag.name === name)) {
        command.flags.push({ name, type: "boolean", required: false });
      }
    }
    command.flags.sort((a, b) => a.name.localeCompare(b.name));
  }
  return `${JSON.stringify(manifest, null, 2)}\n`;
}

function run() {
  const args = process.argv.slice(2);
  if (args.some((arg) => arg !== "--check")) {
    throw new Error("Usage: mise exec -- node docs/scripts/cli-human-output.mjs [--check]");
  }
  const root = fileURLToPath(new URL("../../", import.meta.url));
  const docs = path.join(root, "docs");
  const resources = readResourcePaths(fs.readFileSync(path.join(root, "cmd/api/resources.go"), "utf8"));
  let changed = 0;
  for (const relative of fs.readdirSync(docs, { recursive: true }).sort()) {
    const isManifest = relative === "scripts/manifest.json";
    if (!relative.endsWith(".mdx") && !isManifest) {
      continue;
    }
    const file = path.join(docs, relative);
    const before = fs.readFileSync(file, "utf8");
    const after = isManifest ? migrateManifest(before, resources) : migratePage(before, relative, resources);
    if (after === before) {
      continue;
    }
    changed++;
    console.log(`docs/${relative}`);
    if (!args.includes("--check")) {
      fs.writeFileSync(file, after);
    }
  }
  console.log(`${changed} files ${args.includes("--check") ? "need migration" : "updated"}.`);
  if (args.includes("--check") && changed > 0) {
    process.exitCode = 1;
  }
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  run();
}
