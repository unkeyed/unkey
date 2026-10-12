// Diffs docs/docs.json against the Blume nav model and redirect list.
// Exit 0 only when products, tabs, groups, page ids, and redirects match.

import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const docs = JSON.parse(readFileSync(resolve(root, "docs/docs.json"), "utf8"));
const model = JSON.parse(
  readFileSync(resolve(root, "docsNew/sidebar.json"), "utf8"),
);
const redirects = JSON.parse(
  readFileSync(resolve(root, "docsNew/redirects.json"), "utf8"),
);

type PageNode =
  | string
  | { page: string; route?: string }
  | { group: string; pages: PageNode[] };

const pageId = (page: PageNode): string => {
  if (typeof page === "string") {
    return page;
  }
  if ("page" in page) {
    return page.page;
  }
  return "";
};

const signature = (pages: PageNode[], depth: number): string[] => {
  const lines: string[] = [];
  for (const page of pages) {
    if (typeof page === "object" && page && "group" in page) {
      lines.push(`${"  ".repeat(depth)}G ${page.group}`);
      lines.push(...signature(page.pages, depth + 1));
    } else {
      lines.push(`${"  ".repeat(depth)}P ${pageId(page)}`);
    }
  }
  return lines;
};

const docsLines: string[] = [];
for (const product of docs.navigation.products) {
  docsLines.push(`PRODUCT ${product.product}`);
  for (const tab of product.tabs) {
    docsLines.push(`  TAB ${tab.tab}`);
    docsLines.push(...signature(tab.groups, 2));
  }
}

const blumeLines: string[] = [];
let currentProduct = "";
for (const tab of model.tabs) {
  if (tab.product !== currentProduct) {
    currentProduct = tab.product;
    blumeLines.push(`PRODUCT ${tab.product}`);
  }
  blumeLines.push(`  TAB ${tab.tab}`);
  blumeLines.push(...signature(tab.groups, 2));
}

const navDiffs: string[] = [];
const max = Math.max(docsLines.length, blumeLines.length);
for (let i = 0; i < max; i += 1) {
  if (docsLines[i] !== blumeLines[i]) {
    navDiffs.push(
      `line ${i + 1}\n  docs:  ${docsLines[i] ?? "<missing>"}\n  blume: ${blumeLines[i] ?? "<missing>"}`,
    );
    if (navDiffs.length >= 20) {
      break;
    }
  }
}

type MintRedirect = { source: string; destination: string; permanent?: boolean };
const mintRedirects = docs.redirects as MintRedirect[];
const redirectDiffs: string[] = [];
if (mintRedirects.length !== redirects.length) {
  redirectDiffs.push(
    `count docs=${mintRedirects.length} blume=${redirects.length}`,
  );
}
const count = Math.max(mintRedirects.length, redirects.length);
for (let i = 0; i < count; i += 1) {
  const mint = mintRedirects[i];
  const blume = redirects[i] as
    | { from: string; to: string; status: number }
    | undefined;
  if (!mint || !blume) {
    redirectDiffs.push(`index ${i} missing on one side`);
    continue;
  }
  const status = mint.permanent === false ? 307 : 308;
  if (mint.source !== blume.from || mint.destination !== blume.to || status !== blume.status) {
    redirectDiffs.push(
      `index ${i}\n  docs:  ${mint.source} -> ${mint.destination} (${status})\n  blume: ${blume.from} -> ${blume.to} (${blume.status})`,
    );
  }
  if (redirectDiffs.length >= 20) {
    break;
  }
}

const patterns = mintRedirects.filter(
  (entry) => entry.source.includes(":") || entry.source.includes("*"),
);
const unsupported = patterns.filter((entry) => !entry.source.includes(":slug*"));

console.log("nav lines", docsLines.length, "blume lines", blumeLines.length);
console.log("nav differences", navDiffs.length === 0 ? 0 : navDiffs.length);
for (const diff of navDiffs) {
  console.log(diff);
}
console.log("redirects", mintRedirects.length, "blume", redirects.length);
console.log("redirect differences", redirectDiffs.length === 0 ? 0 : redirectDiffs.length);
for (const diff of redirectDiffs) {
  console.log(diff);
}
console.log("pattern redirects", patterns.length);
console.log("patterns Blume cannot express", unsupported.length);
if (unsupported.length > 0) {
  for (const entry of unsupported) {
    console.log(`  ${entry.source} -> ${entry.destination}`);
  }
}
console.log(
  navDiffs.length === 0 && redirectDiffs.length === 0
    ? "PARITY OK: zero differences"
    : "PARITY FAILED",
);

if (navDiffs.length > 0 || redirectDiffs.length > 0) {
  process.exit(1);
}
