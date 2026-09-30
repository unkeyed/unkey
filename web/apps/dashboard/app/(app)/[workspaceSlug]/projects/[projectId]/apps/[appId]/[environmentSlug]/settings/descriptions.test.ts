import { readFileSync, readdirSync } from "node:fs";
import { join, relative } from "node:path";
import { describe, expect, it } from "vitest";

const MAX_DESCRIPTION_CHARS = 80;
const settingsDir = __dirname;
const FILLER = [/unkey manages it/i, /leave (it )?empty/i, /\be\.g\./i, /for example/i];

function sourceFiles(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) {
      return sourceFiles(path);
    }
    return entry.name.endsWith(".tsx") ? [path] : [];
  });
}

function descriptions(source: string): string[] {
  const direct = [...source.matchAll(/description(?:=|:)\s*"([^"]*)"/g)].map((m) => m[1] ?? "");
  const inBraces = [...source.matchAll(/description=\{([^{}]*)\}/g)].flatMap((m) =>
    [...(m[1] ?? "").matchAll(/"([^"]*)"/g)].map((s) => s[1] ?? ""),
  );
  return [...direct, ...inBraces];
}

const found = sourceFiles(settingsDir).flatMap((file) =>
  descriptions(readFileSync(file, "utf8")).map((text) => ({
    file: relative(settingsDir, file),
    text,
  })),
);

describe("app settings descriptions", () => {
  it("finds the descriptions it checks", () => {
    expect(found.length).toBeGreaterThan(15);
  });

  it(`keeps every description at ${MAX_DESCRIPTION_CHARS} characters or fewer`, () => {
    const tooLong = found
      .filter(({ text }) => text.length > MAX_DESCRIPTION_CHARS)
      .map(({ file, text }) => `${file} (${text.length}): ${text}`);
    expect(tooLong).toEqual([]);
  });

  it("keeps examples, empty-value meanings and filler out of descriptions", () => {
    const filler = found
      .filter(({ text }) => FILLER.some((pattern) => pattern.test(text)))
      .map(({ file, text }) => `${file}: ${text}`);
    expect(filler).toEqual([]);
  });
});
