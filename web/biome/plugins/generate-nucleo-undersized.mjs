import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

// nucleo-ui-outline-12 is a dashboard dependency, so it resolves from there.
const here = dirname(fileURLToPath(import.meta.url));
const pkgDir = join(here, "../../apps/dashboard/node_modules/nucleo-ui-outline-12");
const { version } = JSON.parse(readFileSync(join(pkgDir, "package.json"), "utf8"));
const types = readFileSync(join(pkgDir, "dist/index.d.ts"), "utf8");

const names = [...new Set(types.match(/\bIcon[A-Za-z0-9]+Outline12\b/g) ?? [])]
  .map((name) => name.slice("Icon".length, -"Outline12".length))
  .sort();

if (names.length === 0) {
  throw new Error("no Icon*Outline12 exports found in nucleo-ui-outline-12");
}

const out = `language js

// GENERATED from nucleo-ui-outline-12@${version} by generate-nucleo-undersized.mjs.
// Do not edit by hand; run \`pnpm lint:icons:generate\` after upgrading the package.
//
// An 18px Nucleo glyph squeezed to 12px loses detail, but only the glyphs that
// also ship a 12px drawing have a fix, so the rule lists exactly those names.

pattern nucleo_undersized_18() {
  bubble or {
    \`<$tag $props />\`,
    \`<$tag $props>$_</$tag>\`
  } where {
    $tag <: r"Icon(?:${names.join("|")})Outline18",
    $props <: contains \`className="$cls"\`,
    $cls <: r"(?:^|.*\\s)(?:size-3|size-2\\.5|size-2|size-1\\.5|size-1|size-\\[1[0-2]px\\])(?:\\s.*)?",
    register_diagnostic(span=$tag, message="This glyph has a 12px drawing. At size-3 or smaller import the Outline12 variant from nucleo-ui-outline-12, or render the 18px drawing at size-4 or above.", severity="error")
  }
}

nucleo_undersized_18()
`;

writeFileSync(join(here, "nucleo-undersized-18.grit"), out);
