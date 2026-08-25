import { readFile, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { pathToFileURL } from "node:url";
import { isDeepStrictEqual } from "node:util";

const require = createRequire(import.meta.url);
const { generateMySQLMigration } = require("drizzle-kit/api");

const absent = Symbol("absent");
const identityKeys = new Set(["id", "prevId"]);

export class SnapshotMergeConflict extends Error {
  constructor(path) {
    super(`Conflicting schema changes at ${path.join(".")}`);
    this.name = "SnapshotMergeConflict";
  }
}

function isRecord(value) {
  return value !== absent && value !== null && typeof value === "object" && !Array.isArray(value);
}

function same(left, right) {
  if (left === absent || right === absent) {
    return left === right;
  }
  return isDeepStrictEqual(left, right);
}

function clone(value) {
  return value === absent ? absent : structuredClone(value);
}

// Merge the pull request's base-to-head change into the current database value.
// A value changed differently on both sides is a conflict rather than a guess.
function mergeValue(base, head, current, path) {
  if (same(head, base)) {
    return clone(current);
  }
  if (same(current, base) || same(current, head)) {
    return clone(head);
  }
  if (base === absent || head === absent || current === absent) {
    throw new SnapshotMergeConflict(path);
  }
  if (!isRecord(base) || !isRecord(head) || !isRecord(current)) {
    throw new SnapshotMergeConflict(path);
  }

  const merged = {};
  const keys = new Set([...Object.keys(base), ...Object.keys(head), ...Object.keys(current)]);
  for (const key of keys) {
    const value = mergeValue(
      Object.hasOwn(base, key) ? base[key] : absent,
      Object.hasOwn(head, key) ? head[key] : absent,
      Object.hasOwn(current, key) ? current[key] : absent,
      [...path, key],
    );
    if (value !== absent) {
      merged[key] = value;
    }
  }
  return merged;
}

export function mergeSnapshots(base, head, current) {
  for (const key of ["version", "dialect"]) {
    if (!same(base[key], head[key]) || !same(base[key], current[key])) {
      throw new SnapshotMergeConflict([key]);
    }
  }

  const merged = structuredClone(current);
  const keys = new Set([...Object.keys(base), ...Object.keys(head), ...Object.keys(current)]);
  for (const key of keys) {
    if (identityKeys.has(key) || key === "version" || key === "dialect") {
      continue;
    }
    const value = mergeValue(
      Object.hasOwn(base, key) ? base[key] : absent,
      Object.hasOwn(head, key) ? head[key] : absent,
      Object.hasOwn(current, key) ? current[key] : absent,
      [key],
    );
    if (value === absent) {
      delete merged[key];
    } else {
      merged[key] = value;
    }
  }
  return merged;
}

export async function generateEffectiveStatements(base, head, current) {
  const merged = mergeSnapshots(base, head, current);
  if (isDeepStrictEqual(current, merged)) {
    return [];
  }

  const statements = await generateMySQLMigration(current, merged);
  if (statements.length === 0) {
    throw new Error("Drizzle generated no SQL for a non-empty merged schema");
  }
  return statements;
}

async function main() {
  const [basePath, headPath, currentPath, outputPath] = process.argv.slice(2);
  if (!basePath || !headPath || !currentPath || !outputPath) {
    throw new Error("Usage: merge-mysql-snapshots <base> <head> <current> <output>");
  }

  const [base, head, current] = await Promise.all(
    [basePath, headPath, currentPath].map(async (path) => JSON.parse(await readFile(path, "utf8"))),
  );
  const statements = await generateEffectiveStatements(base, head, current);
  await writeFile(outputPath, statements.map((statement) => statement.trim()).join("\n--> statement-breakpoint\n"));
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  await main();
}
