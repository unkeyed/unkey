import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { resolve } from "node:path";
import { isDeepStrictEqual } from "node:util";
import mysql, { type RowDataPacket } from "mysql2/promise";
import { z } from "zod";

const definitions = z.record(z.string(), z.unknown());
const metadata = z.record(z.string(), z.looseObject({ columns: definitions }).optional());

export const snapshotSchema = z.looseObject({
  version: z.literal("5"),
  dialect: z.literal("mysql"),
  tables: z.record(
    z.string(),
    z.looseObject({
      columns: z.record(
        z.string(),
        z.looseObject({ type: z.string(), default: z.unknown().optional() }),
      ),
      indexes: z.record(
        z.string(),
        z.looseObject({
          columns: z.array(z.string()),
          isUnique: z.boolean(),
          using: z.string().optional(),
          algorithm: z.string().optional(),
          lock: z.string().optional(),
        }),
      ),
      foreignKeys: definitions,
      compositePrimaryKeys: definitions,
      uniqueConstraints: definitions,
      checkConstraint: definitions,
    }),
  ),
  views: definitions,
  internal: z.looseObject({ tables: metadata.optional(), indexes: metadata.optional() }).optional(),
});

type Snapshot = z.infer<typeof snapshotSchema>;

const kit = z
  .object({ generateMySQLMigration: z.unknown() })
  .parse(createRequire(resolve("package.json"))("drizzle-kit/api"));
if (typeof kit.generateMySQLMigration !== "function") {
  throw new Error("drizzle-kit/api does not expose generateMySQLMigration");
}
const generate = kit.generateMySQLMigration;

async function generateMySQLMigration(previous: Snapshot, current: Snapshot): Promise<string[]> {
  const statements: unknown = await generate(previous, current);
  return z.array(z.string()).parse(statements);
}

function checkRenames(base: Snapshot, head: Snapshot): void {
  checkNames(base.tables, head.tables, "tables");
  checkNames(base.views, head.views, "views");
  for (const [name, table] of Object.entries(base.tables)) {
    const next = head.tables[name];
    if (next) {
      checkNames(table.columns, next.columns, `${name}.columns`);
    }
  }
}

function checkNames(before: object, after: object, path: string): void {
  const removed = Object.keys(before).filter(
    (key) => !Object.prototype.hasOwnProperty.call(after, key),
  );
  const added = Object.keys(after).filter(
    (key) => !Object.prototype.hasOwnProperty.call(before, key),
  );
  if (removed.length && added.length) {
    throw new Error(
      `Possible rename in ${path}: removed [${removed}], added [${added}]. PlanetScale deploy requests do not support direct renames. Add the new column/table first, copy data and switch readers, then drop the old one in a separate deploy request. Split unrelated additions and removals too; this task never guesses rename intent.`,
    );
  }
}

function mergeEntries<T>(
  base: Record<string, T>,
  head: Record<string, T>,
  live: Record<string, T>,
  path: string,
): Record<string, T> {
  const result = { ...live };
  for (const key of new Set([...Object.keys(base), ...Object.keys(head)])) {
    if (isDeepStrictEqual(base[key], head[key]) || isDeepStrictEqual(live[key], head[key])) {
      continue;
    }
    if (!isDeepStrictEqual(live[key], base[key])) {
      throw new Error(
        `Schema conflict at ${path}.${key}: the database matches neither the Git base nor the proposed definition. No SQL was applied. Reconcile this object before retrying.`,
      );
    }
    if (Object.prototype.hasOwnProperty.call(head, key)) {
      result[key] = head[key];
    } else {
      delete result[key];
    }
  }
  return result;
}

export async function planBranchChanges(
  baseSnapshot: Snapshot,
  headSnapshot: Snapshot,
  liveSnapshot: Snapshot,
): Promise<string[]> {
  const base = normalizeSnapshot(baseSnapshot);
  const head = normalizeSnapshot(headSnapshot);
  const live = normalizeSnapshot(liveSnapshot);
  checkRenames(base, head);
  const tables = { ...live.tables };
  for (const name of new Set([...Object.keys(base.tables), ...Object.keys(head.tables)])) {
    const before = base.tables[name];
    const after = head.tables[name];
    const current = live.tables[name];
    if (isDeepStrictEqual(before, after)) {
      continue;
    }
    if (!before || !after || !current) {
      const merged = mergeEntries(
        before ? { [name]: before } : {},
        after ? { [name]: after } : {},
        current ? { [name]: current } : {},
        "tables",
      );
      if (merged[name]) {
        tables[name] = merged[name];
      } else {
        delete tables[name];
      }
      continue;
    }
    tables[name] = {
      ...current,
      columns: mergeEntries(before.columns, after.columns, current.columns, `${name}.columns`),
      indexes: mergeEntries(before.indexes, after.indexes, current.indexes, `${name}.indexes`),
      foreignKeys: mergeEntries(
        before.foreignKeys,
        after.foreignKeys,
        current.foreignKeys,
        `${name}.foreignKeys`,
      ),
      compositePrimaryKeys: mergeEntries(
        before.compositePrimaryKeys,
        after.compositePrimaryKeys,
        current.compositePrimaryKeys,
        `${name}.primaryKeys`,
      ),
      uniqueConstraints: mergeEntries(
        before.uniqueConstraints,
        after.uniqueConstraints,
        current.uniqueConstraints,
        `${name}.uniqueConstraints`,
      ),
      checkConstraint: mergeEntries(
        before.checkConstraint,
        after.checkConstraint,
        current.checkConstraint,
        `${name}.checks`,
      ),
    };
  }
  const desired = {
    ...live,
    tables,
    views: mergeEntries(base.views, head.views, live.views, "views"),
    internal: {
      tables: mergeMetadata(base.internal?.tables, head.internal?.tables, live.internal?.tables),
      indexes: mergeMetadata(
        base.internal?.indexes,
        head.internal?.indexes,
        live.internal?.indexes,
      ),
    },
  };
  checkRenames(live, desired);
  return generateMySQLMigration(live, desired);
}

function normalizeSnapshot(input: Snapshot): Snapshot {
  const snapshot = snapshotSchema.parse(JSON.parse(JSON.stringify(input)));
  for (const table of Object.values(snapshot.tables)) {
    for (const column of Object.values(table.columns)) {
      if (column.type === "boolean") {
        column.type = "tinyint(1)";
        if (typeof column.default === "boolean") {
          column.default = Number(column.default);
        }
      }
    }
    for (const [name, index] of Object.entries(table.indexes)) {
      if (index.isUnique && !index.using && !index.algorithm && !index.lock) {
        table.uniqueConstraints[name] = { name, columns: index.columns };
        delete table.indexes[name];
      }
    }
  }
  return snapshot;
}

function mergeMetadata<T>(
  base: Record<string, { columns: Record<string, T> } | undefined> = {},
  head: Record<string, { columns: Record<string, T> } | undefined> = {},
  live: Record<string, { columns: Record<string, T> } | undefined> = {},
) {
  const result = { ...live };
  for (const name of new Set([...Object.keys(base), ...Object.keys(head)])) {
    result[name] = {
      columns: mergeEntries(
        base[name]?.columns ?? {},
        head[name]?.columns ?? {},
        live[name]?.columns ?? {},
        `metadata.${name}`,
      ),
    };
  }
  return result;
}

async function readSnapshot(path: string): Promise<Snapshot> {
  const snapshot = snapshotSchema.parse(JSON.parse(readFileSync(path, "utf8")));
  await generateMySQLMigration(snapshot, snapshot);
  return snapshot;
}

interface ColumnCollation extends RowDataPacket {
  tableName: string;
  columnName: string;
  collation: string | null;
}

async function readCollations(
  connection: mysql.Connection,
  base: Snapshot,
  head: Snapshot,
  live: Snapshot,
): Promise<void> {
  const [columns] = await connection.query<ColumnCollation[]>(
    "SELECT TABLE_NAME AS tableName, COLUMN_NAME AS columnName, COLLATION_NAME AS collation " +
      "FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE()",
  );
  for (const { tableName, columnName, collation } of columns) {
    const expected = [
      base.tables[tableName]?.columns[columnName],
      head.tables[tableName]?.columns[columnName],
    ];
    const column = live.tables[tableName]?.columns[columnName];
    if (column && collation && expected.some((value) => / COLLATE /i.test(value?.type ?? ""))) {
      column.type = `${column.type} COLLATE ${collation}`;
    }
  }
}

async function main(): Promise<void> {
  const [, , basePath, headPath, livePath] = process.argv;
  if (!basePath || !headPath) {
    throw new Error(
      "Expected base and head snapshot paths, followed by a live snapshot or --check.",
    );
  }
  const base = await readSnapshot(basePath);
  const head = await readSnapshot(headPath);
  checkRenames(base, head);
  if (livePath === "--check") {
    return;
  }
  if (!livePath || !process.env.DRIZZLE_DATABASE_URL) {
    throw new Error("Expected live snapshot path and DRIZZLE_DATABASE_URL.");
  }
  const live = await readSnapshot(livePath);
  const connection = await mysql.createConnection(process.env.DRIZZLE_DATABASE_URL);
  try {
    await readCollations(connection, base, head, live);
    const statements = await planBranchChanges(base, head, live);
    if (statements.length === 0) {
      console.info("This branch's schema changes are already applied; no SQL to execute.");
      return;
    }
    console.info(statements.join("\n"));
    for (const statement of statements) {
      await connection.query(statement);
    }
    console.info(`Applied ${statements.length} statement(s) from this Git branch only.`);
  } finally {
    await connection.end();
  }
}

if (process.argv[1] === resolve("scripts/pscale-branch.mts")) {
  main().catch((error: unknown) => {
    console.error(error instanceof Error ? error.message : "PlanetScale branch migration failed.");
    process.exitCode = 1;
  });
}
