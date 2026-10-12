// Copies Mintlify content from ../docs into this package and writes the
// Blume nav, redirect, and OpenAPI inputs. Re-runnable. Does not edit ../docs.

import { spawnSync } from "node:child_process";
import {
  cpSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import YAML from "yaml";

const here = path.dirname(fileURLToPath(import.meta.url));
const pkg = path.resolve(here, "..");
const repo = path.resolve(pkg, "..");
const source = path.join(repo, "docs");
const content = path.join(pkg, "docs");
const publicDir = path.join(pkg, "public");
const ICONS = JSON.parse(readFileSync(path.join(here, "icons.json"), "utf8"));

const NON_SLUG = /[^\p{L}\p{M}\p{N}]+/gu;
const FORMAT_CHARS = /\p{Cf}/gu;
const CASE_BOUNDARY = /(?<before>[\p{Ll}\p{N}])(?<capital>\p{Lu})/gu;
const ACRONYM_BOUNDARY = /(?<acronym>\p{Lu})(?<word>\p{Lu}\p{Ll})/gu;

const trimChar = (value, char) => {
  let start = 0;
  let end = value.length;
  while (start < end && value[start] === char) start += 1;
  while (end > start && value[end - 1] === char) end -= 1;
  return value.slice(start, end);
};

const slugify = (text) =>
  trimChar(
    text.normalize("NFC").toLowerCase().replace(FORMAT_CHARS, "").replace(NON_SLUG, "-"),
    "-",
  );

const splitIdentifier = (identifier) =>
  identifier.replace(CASE_BOUNDARY, "$<before>-$<capital>").replace(ACRONYM_BOUNDARY, "$<acronym>-$<word>");

const tagSlug = (name) => slugify(splitIdentifier(name));

const mintSlug = (summary, operationId) => {
  const raw = (summary || operationId || "operation").replace(/\(([^)]*)\)/g, "$1");
  return slugify(raw) || "operation";
};

const CALLS = {
  Check: "success",
  Danger: "danger",
  Error: "danger",
  Info: "info",
  Note: "note",
  Tip: "tip",
  Warning: "warning",
};

const COMPONENT_NAMES = [
  "Accordion",
  "AccordionGroup",
  "Card",
  "CardGroup",
  "Check",
  "CodeGroup",
  "Column",
  "Columns",
  "Danger",
  "Error",
  "Expandable",
  "Frame",
  "Icon",
  "Info",
  "Note",
  "Panel",
  "ParamField",
  "RequestExample",
  "RequestField",
  "ResponseExample",
  "ResponseField",
  "Step",
  "Steps",
  "Tab",
  "Tabs",
  "Tip",
  "Tooltip",
  "Update",
  "View",
  "Warning",
];

const walkFiles = (dir, out = []) => {
  for (const name of readdirSync(dir)) {
    const full = path.join(dir, name);
    if (statSync(full).isDirectory()) {
      walkFiles(full, out);
    } else {
      out.push(full);
    }
  }
  return out;
};

const splitFences = (text) => {
  const parts = [];
  const re = /(^|\n)(```+)[^\n]*\n[\s\S]*?\n\2(?=\n|$)/g;
  let cursor = 0;
  for (const match of text.matchAll(re)) {
    const index = match.index + (match[1] === "\n" ? 1 : 0);
    if (index > cursor) {
      parts.push({ fence: false, text: text.slice(cursor, index) });
    }
    const fenceText = text.slice(index, index + match[0].length - (match[1] === "\n" ? 1 : 0));
    parts.push({ fence: true, text: match[1] === "\n" && index !== match.index ? `\n${fenceText}` : fenceText });
    cursor = index + fenceText.length;
  }
  if (cursor < text.length) {
    parts.push({ fence: false, text: text.slice(cursor) });
  }
  return parts;
};

const remapIcons = (text) =>
  text.replace(/\bicon="([^"]+)"/g, (full, name) => {
    const key = name.toLowerCase();
    if (!(key in ICONS)) {
      return full;
    }
    if (ICONS[key] === null) {
      return "";
    }
    return `icon="${ICONS[key]}"`;
  });

const calloutToDirective = (text) =>
  text.replace(
    /<(Note|Tip|Warning|Info|Check|Danger|Error)(\s[^>]*)?>([\s\S]*?)<\/\1>/g,
    (_full, name, attrs = "", body) => {
      const title = /title="([^"]*)"/.exec(attrs)?.[1];
      const kind = CALLS[name];
      const head = title ? `:::${kind}[${title}]` : `:::${kind}`;
      return `${head}\n${body.trim()}\n:::`;
    },
  );

const transformProse = (text, file) => {
  let next = text.replace(/^import\s.+$/gm, "");
  next = next.replace(/\{versions\.([A-Za-z0-9_]+)\}/g, "{{$1}}");
  next = calloutToDirective(next);
  next = next.replaceAll("<AccordionGroup>", "%%AG_OPEN%%");
  next = next.replaceAll("</AccordionGroup>", "%%AG_CLOSE%%");
  next = next.replaceAll("<Accordion ", "<AccordionItem ");
  next = next.replaceAll("<Accordion>", "<AccordionItem>");
  next = next.replaceAll("</Accordion>", "</AccordionItem>");
  next = next.replaceAll("%%AG_OPEN%%", "<Accordion>");
  next = next.replaceAll("%%AG_CLOSE%%", "</Accordion>");
  next = next.replace(/<Tabs(?![^>]*\binline\b)/g, "<Tabs inline");
  next = remapIcons(next);
  next = next.replace(/\bclassName=/g, "class=");
  next = escapeProseBraces(next);
  const rel = (snippet) => {
    let relative = path.relative(path.dirname(file), path.join(content, "_snippets", snippet));
    if (!relative.startsWith(".")) {
      relative = `./${relative}`;
    }
    return relative.split(path.sep).join("/");
  };
  next = next.replaceAll("<CliInstall />", `<include>${rel("cli-install.mdx")}</include>`);
  next = next.replaceAll("<RootKeyPrereq />", `<include>${rel("root-key-prereq.mdx")}</include>`);
  return next;
};

const escapeProseBraces = (text) => {
  let out = "";
  let index = 0;
  while (index < text.length) {
    const variable = /^\{\{[A-Za-z0-9_-]+\}\}/.exec(text.slice(index));
    if (variable) {
      out += variable[0];
      index += variable[0].length;
      continue;
    }
    const next = text[index + 1] ?? "";
    if (text[index] === "<" && /[A-Za-z/!]/.test(next)) {
      const end = text.indexOf(">", index);
      if (end === -1) {
        out += text.slice(index);
        break;
      }
      out += text.slice(index, end + 1);
      index = end + 1;
      continue;
    }
    if (text[index] === "{" || text[index] === "}") {
      out += `\\${text[index]}`;
      index += 1;
      continue;
    }
    out += text[index];
    index += 1;
  }
  return out;
};

const transformMdx = (text, file) => {
  let body = text;
  let frontmatter = "";
  if (text.startsWith("---\n")) {
    const end = text.indexOf("\n---\n", 4);
    if (end !== -1) {
      frontmatter = text.slice(0, end + 5);
      body = text.slice(end + 5);
    }
  }
  const transformed = splitFences(body)
    .map((part) => (part.fence ? part.text : transformProse(part.text, file)))
    .join("");
  return `${frontmatter}${transformed}`;
};

const dropSources = (text) => {
  if (!text.startsWith("---\n")) {
    return text;
  }
  const end = text.indexOf("\n---\n", 4);
  if (end === -1) {
    return text;
  }
  const fm = text.slice(4, end).split("\n");
  const kept = [];
  let skipping = false;
  for (const line of fm) {
    if (!/^\s/.test(line) && line !== "") {
      skipping = line.startsWith("sources:");
    }
    if (!skipping) {
      kept.push(line);
    }
  }
  return `---\n${kept.join("\n").replace(/\n+$/u, "")}\n---\n${text.slice(end + 5)}`;
};

const countComponents = () => {
  const counts = Object.fromEntries(COMPONENT_NAMES.map((name) => [name, 0]));
  const files = walkFiles(source).filter((file) => file.endsWith(".mdx"));
  const patterns = Object.fromEntries(
    COMPONENT_NAMES.map((name) => [
      name,
      new RegExp(`<${name}(?=[\\s>/])`, "g"),
    ]),
  );
  for (const file of files) {
    const text = readFileSync(file, "utf8");
    for (const name of COMPONENT_NAMES) {
      counts[name] += text.match(patterns[name])?.length ?? 0;
    }
  }
  counts.snippetImports = 0;
  for (const file of files) {
    const text = readFileSync(file, "utf8");
    counts.snippetImports += text.match(/^import\s/gm)?.length ?? 0;
  }
  return { files: files.length, counts };
};

const loadSpec = () => YAML.parse(readFileSync(path.join(pkg, "openapi/unkey.yaml"), "utf8"));

const indexSpec = (spec) => {
  const byPath = new Map();
  const byId = new Map();
  for (const [opPath, item] of Object.entries(spec.paths ?? {})) {
    for (const [method, operation] of Object.entries(item)) {
      if (!operation || typeof operation !== "object" || !operation.operationId) {
        continue;
      }
      const record = {
        method: method.toUpperCase(),
        operation,
        path: opPath,
        tag: operation.tags?.[0] ?? "operations",
      };
      byPath.set(`${record.method} ${opPath}`, record);
      if (!byId.has(operation.operationId)) {
        byId.set(operation.operationId, record);
      }
    }
  }
  return { byId, byPath };
};

const resolveOperation = (entry, index) => {
  const match = /^(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+(\S+)$/.exec(entry);
  if (!match) {
    return null;
  }
  const method = match[1];
  let opPath = match[2].startsWith("/") ? match[2] : `/${match[2]}`;
  return (
    index.byPath.get(`${method} ${opPath}`) ??
    index.byPath.get(`${method} ${opPath.replace("/v3/", "/v2/")}`) ??
    index.byId.get(opPath.split("/").pop()) ??
    null
  );
};

const slugFor = (record, used, directory) => {
  const tag = record.tag;
  let slug = mintSlug(record.operation.summary, record.operation.operationId);
  const bucket = used.get(`${directory}:${tag}`) ?? new Set();
  used.set(`${directory}:${tag}`, bucket);
  if (bucket.has(slug)) {
    let n = 2;
    while (bucket.has(`${slug}-${n}`)) n += 1;
    slug = `${slug}-${n}`;
  }
  bucket.add(slug);
  return { slug, tag, tagSlug: tagSlug(tag) };
};

const filterSpec = (spec, records) => {
  const copy = structuredClone(spec);
  const keep = new Set(records.map((record) => `${record.method} ${record.path}`));
  for (const [opPath, item] of Object.entries(copy.paths ?? {})) {
    for (const method of Object.keys(item)) {
      const operation = item[method];
      if (!operation || typeof operation !== "object" || !operation.operationId) {
        continue;
      }
      const key = `${method.toUpperCase()} ${opPath}`;
      if (!keep.has(key)) {
        delete item[method];
        continue;
      }
      const record = records.find((itemRecord) => `${itemRecord.method} ${itemRecord.path}` === key);
      operation.operationId = record.slug;
    }
    const methods = Object.keys(item).filter((key) =>
      ["get", "post", "put", "patch", "delete", "head", "options"].includes(key),
    );
    if (methods.length === 0) {
      delete copy.paths[opPath];
    }
  }
  return copy;
};

const endpointPages = (pages, directory, index, used, missing) =>
  pages.map((page) => {
    if (page && typeof page === "object" && page.group) {
      return {
        group: page.group,
        pages: endpointPages(page.pages, directory, index, used, missing),
      };
    }
    const record = resolveOperation(page, index);
    if (!record) {
      missing.push(page);
      return { page, route: `/${directory}/missing` };
    }
    const slug = slugFor(record, used, directory);
    record.slug = slug.slug;
    return {
      page,
      route: `/${directory}/${slug.tagSlug}/${slug.slug}`,
      spec: record,
    };
  });

const collectRecords = (pages, out) => {
  for (const page of pages) {
    if (page && typeof page === "object" && page.group) {
      collectRecords(page.pages, out);
    } else if (page?.spec) {
      out.push({
        method: page.spec.method,
        operation: page.spec.operation,
        path: page.spec.path,
        slug: page.spec.slug,
        tag: page.spec.tag,
      });
    }
  }
};

const firstRoute = (pages) => {
  for (const page of pages) {
    if (page && typeof page === "object" && page.group) {
      const nested = firstRoute(page.pages);
      if (nested) {
        return nested;
      }
    } else if (typeof page === "string") {
      if (page === "index") return "/";
      if (page.endsWith("/index")) return `/${page.slice(0, -"/index".length)}`;
      return `/${page}`;
    } else if (page?.route) {
      return page.route;
    }
  }
  return "/";
};

const buildModel = (docs, spec) => {
  const index = indexSpec(spec);
  const missing = [];
  const tabs = [];
  const specRecords = new Map();
  const used = new Map();
  for (const product of docs.navigation.products) {
    for (const tab of product.tabs) {
      const directory = tab.openapi?.directory;
      const groups = directory
        ? tab.groups.map((group) => ({
            group: group.group,
            pages: endpointPages(group.pages, directory, index, used, missing),
          }))
        : tab.groups;
      if (directory) {
        const records = [];
        collectRecords(groups, records);
        specRecords.set(directory, records);
      }
      const pathPrefix = directory
        ? `/${directory}`
        : product.product === "Platform" && tab.tab === "Guides"
          ? "/"
          : `/${commonDirectory(tab.groups)}`;
      const href = pathPrefix === "/" ? "/" : pathPrefix;
      tabs.push({
        groups,
        href,
        icon: ICONS[tab.icon] ?? tab.icon,
        path: pathPrefix,
        product: product.product,
        tab: tab.tab,
        wrap: !(product.product === "Platform" && tab.tab === "Guides"),
      });
    }
  }
  const strip = (pages) => {
    for (const page of pages) {
      if (page?.group) {
        strip(page.pages);
      } else if (page && typeof page === "object") {
        delete page.spec;
      }
    }
  };
  for (const tab of tabs) {
    strip(tab.groups);
  }
  return { missing, specRecords, tabs };
};

const flatIds = (pages, out = []) => {
  for (const page of pages) {
    if (typeof page === "string") {
      out.push(page);
    } else if (page?.group) {
      flatIds(page.pages, out);
    }
  }
  return out;
};

const commonDirectory = (groups) => {
  const ids = groups.flatMap((group) => flatIds(group.pages));
  const parts = ids.map((id) => id.split("/"));
  if (parts.length === 0) {
    return "";
  }
  let index = 0;
  while (parts.every((part) => part[index] && part[index] === parts[0][index])) {
    index += 1;
  }
  return parts[0].slice(0, index).join("/");
};

const copyContent = () => {
  rmSync(content, { force: true, recursive: true });
  mkdirSync(content, { recursive: true });
  const files = walkFiles(source).filter((file) => file.endsWith(".mdx") || file.endsWith(".md"));
  for (const file of files) {
    const rel = path.relative(source, file);
    if (rel.startsWith(`scripts${path.sep}`) || rel === "docs.json") {
      continue;
    }
    const destRel = rel.startsWith(`snippets${path.sep}`)
      ? path.join("_snippets", path.relative("snippets", rel))
      : rel;
    if (destRel.endsWith(`${path.sep}versions.mdx`) || destRel === "versions.mdx" || destRel.endsWith("versions.mdx")) {
      continue;
    }
    const dest = path.join(content, destRel);
    mkdirSync(path.dirname(dest), { recursive: true });
    const transformed = dropSources(transformMdx(readFileSync(file, "utf8"), dest));
    writeFileSync(dest, transformed);
  }
};

const copyAssets = () => {
  mkdirSync(publicDir, { recursive: true });
  const files = walkFiles(source).filter((file) => !file.endsWith(".mdx") && !file.endsWith(".md"));
  for (const file of files) {
    const rel = path.relative(source, file);
    if (
      rel.startsWith(`scripts${path.sep}`) ||
      rel.startsWith(`snippets${path.sep}`) ||
      rel === "docs.json" ||
      rel === "style.css" ||
      rel === "card-illustrations.css"
    ) {
      continue;
    }
    const dest = path.join(publicDir, rel);
    mkdirSync(path.dirname(dest), { recursive: true });
    cpSync(file, dest);
  }
  cpSync(path.join(source, "favicon-light.svg"), path.join(publicDir, "favicon.svg"));
  cpSync(path.join(source, "favicon-dark.svg"), path.join(publicDir, "favicon-dark.svg"));
  const logo = readFileSync("/tmp/marketing-site/public/logo-dark.svg", "utf8").replace(
    /fill="(?:#040406|white)"/g,
    'fill="currentColor"',
  );
  writeFileSync(path.join(publicDir, "logo.svg"), logo);
};

const versions = () => {
  const text = readFileSync(path.join(source, "snippets/versions.mdx"), "utf8");
  const vars = {};
  for (const match of text.matchAll(/(\w+):\s*"([^"]+)"/g)) {
    vars[match[1]] = match[2];
  }
  return vars;
};

const redirectsFrom = (docs) =>
  docs.redirects.map((entry) => ({
    from: entry.source,
    status: entry.permanent === false ? 307 : 308,
    to: entry.destination,
  }));

const inventory = countComponents();
writeFileSync(path.join(pkg, "feature-inventory.json"), `${JSON.stringify(inventory, null, 2)}\n`);

copyContent();
copyAssets();

const sectionIndexes = [
  ["compute/cli/index.mdx", "CLI", "Compute commands for projects, apps, deployments, and the gateway."],
  ["api-management/cli/index.mdx", "CLI", "API management commands for keys, identities, permissions, and rate limits."],
  ["api-management/sdks/index.mdx", "SDKs", "Official and community SDKs for the API management API."],
];
for (const [rel, title, description] of sectionIndexes) {
  const dest = path.join(content, rel);
  mkdirSync(path.dirname(dest), { recursive: true });
  writeFileSync(
    dest,
    `---\ntitle: ${title}\ndescription: "${description}"\nsidebar:\n  hidden: true\n---\n\nUse the sidebar to open a page in this section.\n`,
  );
}

const codemod = spawnSync(
  process.execPath,
  [path.join(here, "mintlify-codemod.mjs"), "--write", content],
  { encoding: "utf8" },
);
if (codemod.status !== 0) {
  console.error(codemod.stdout, codemod.stderr);
  process.exit(codemod.status ?? 1);
}

const docs = JSON.parse(readFileSync(path.join(source, "docs.json"), "utf8"));
const spec = loadSpec();
const { missing, specRecords, tabs } = buildModel(docs, spec);
if (missing.length > 0) {
  console.error("unresolved openapi nav entries", missing);
  process.exit(1);
}

for (const [directory, records] of specRecords) {
  const filtered = filterSpec(spec, records);
  const name = directory.startsWith("platform")
    ? "platform.yaml"
    : directory.startsWith("compute")
      ? "compute.yaml"
      : "api-management.yaml";
  writeFileSync(path.join(pkg, "openapi", name), YAML.stringify(filtered));
}

const selectors = [
  {
    items: docs.navigation.products.map((product) => ({
      description: product.description,
      icon: ICONS[product.icon] ?? product.icon,
      label: product.name,
      path:
        product.product === "Platform"
          ? "/"
          : product.product === "Compute"
            ? "/compute"
            : "/api-management",
    })),
    kind: "product",
    label: "Product",
  },
];

writeFileSync(
  path.join(pkg, "sidebar.json"),
  `${JSON.stringify({ selectors, tabs }, null, 2)}\n`,
);
writeFileSync(
  path.join(pkg, "redirects.json"),
  `${JSON.stringify(redirectsFrom(docs), null, 2)}\n`,
);
writeFileSync(path.join(pkg, "variables.json"), `${JSON.stringify(versions(), null, 2)}\n`);

const routes = new Set();
const walkRoutes = (pages) => {
  for (const page of pages) {
    if (page && typeof page === "object" && page.group) {
      walkRoutes(page.pages);
    } else if (page && typeof page === "object" && page.route) {
      routes.add(page.route);
    } else if (typeof page === "string") {
      routes.add(page === "index" ? "/" : page.endsWith("/index") ? `/${page.slice(0, -6)}` : `/${page}`);
    }
  }
};
for (const tab of tabs) {
  walkRoutes(tab.groups);
  if (tab.path !== "/") {
    routes.add(tab.path);
  }
}
const missed = docs.redirects.filter(
  (entry) => !entry.destination.includes(":") && !entry.destination.includes("*") && !routes.has(entry.destination),
);
const aliases = missed.flatMap((entry) => {
  if (!entry.destination.endsWith("/index")) {
    return [];
  }
  const canonical = entry.destination.slice(0, -"/index".length) || "/";
  if (!routes.has(canonical)) {
    return [];
  }
  return [{ from: entry.destination, status: 308, to: canonical }];
});
const aliasSeen = new Set();
const uniqueAliases = aliases.filter((entry) => {
  if (aliasSeen.has(entry.from)) {
    return false;
  }
  aliasSeen.add(entry.from);
  return true;
});
writeFileSync(path.join(pkg, "alias-redirects.json"), `${JSON.stringify(uniqueAliases, null, 2)}\n`);
writeFileSync(
  path.join(pkg, "redirect-target-gaps.json"),
  `${JSON.stringify(
    missed.filter((entry) => !aliasSeen.has(entry.destination)),
    null,
    2,
  )}\n`,
);
console.log("content copied");
console.log("openapi nav entries unresolved", missing.length);
console.log("exact redirect destinations not in the nav", missed.length);
console.log("tabs", tabs.length, "redirects", docs.redirects.length);
