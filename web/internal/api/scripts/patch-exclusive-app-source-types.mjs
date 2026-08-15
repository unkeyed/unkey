import { readFile, writeFile } from "node:fs/promises";

const replacements = {
  "v2appscreateapprequestbodyunion.ts": [
    ["git?: any | undefined;", "git?: never | undefined;", 2],
    ["docker?: any | undefined;", "docker?: never | undefined;", 2],
    ["git: z.any().optional(),", "git: z.never().optional(),", 1],
    ["docker: z.any().optional(),", "docker: z.never().optional(),", 1],
  ],
  "v2appsupdateapprequestbodyunion.ts": [
    ["docker?: any | undefined;", "docker?: never | undefined;", 8],
    ["name?: any | undefined;", "name?: never | undefined;", 2],
    ["slug?: any | undefined;", "slug?: never | undefined;", 2],
    ["git?: any | undefined;", "git?: never | undefined;", 2],
    ["deleteProtection?: any | undefined;", "deleteProtection?: never | undefined;", 2],
    ["docker: z.any().optional(),", "docker: z.never().optional(),", 4],
    ["name: z.any().optional(),", "name: z.never().optional(),", 1],
    ["slug: z.any().optional(),", "slug: z.never().optional(),", 1],
    ["git: z.any().optional(),", "git: z.never().optional(),", 1],
    [
      "deleteProtection: z.any().optional(),",
      "deleteProtection: z.never().optional(),",
      1,
    ],
  ],
};

for (const [filename, fileReplacements] of Object.entries(replacements)) {
  const path = new URL(`../src/models/components/${filename}`, import.meta.url);
  let source = await readFile(path, "utf8");

  for (const [generated, corrected, expectedCount] of fileReplacements) {
    const actualCount = source.split(generated).length - 1;
    if (actualCount !== expectedCount) {
      throw new Error(
        `${filename}: expected ${expectedCount} occurrences of ${JSON.stringify(generated)}, found ${actualCount}`,
      );
    }
    source = source.replaceAll(generated, corrected);
  }

  await writeFile(path, source);
}
