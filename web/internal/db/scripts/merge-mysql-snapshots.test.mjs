import assert from "node:assert/strict";
import test from "node:test";
import {
  SnapshotMergeConflict,
  generateEffectiveStatements,
  mergeSnapshots,
} from "./merge-mysql-snapshots.mjs";

function column(name, type = "int") {
  return {
    name,
    type,
    primaryKey: false,
    notNull: true,
    autoincrement: false,
  };
}

function snapshot() {
  return {
    version: "5",
    dialect: "mysql",
    id: "00000000-0000-0000-0000-000000000001",
    prevId: "00000000-0000-0000-0000-000000000000",
    tables: {
      users: {
        name: "users",
        columns: { id: column("id") },
        indexes: {},
        foreignKeys: {},
        compositePrimaryKeys: {},
        uniqueConstraints: {},
        checkConstraint: {},
      },
    },
    views: {},
    _meta: { schemas: {}, tables: {}, columns: {} },
    internal: { tables: {}, indexes: {} },
  };
}

test("preserves unrelated current changes", () => {
  const base = snapshot();
  const head = structuredClone(base);
  head.tables.users.columns.name = column("name", "varchar(255)");
  const current = structuredClone(base);
  current.tables.users.indexes.id_idx = {
    name: "id_idx",
    columns: ["id"],
    isUnique: false,
  };

  const merged = mergeSnapshots(base, head, current);

  assert.deepEqual(merged.tables.users.columns.name, head.tables.users.columns.name);
  assert.deepEqual(merged.tables.users.indexes.id_idx, current.tables.users.indexes.id_idx);
});

test("treats an applied pull request change as a no-op", async () => {
  const base = snapshot();
  const head = structuredClone(base);
  head.tables.users.columns.name = column("name", "varchar(255)");
  const current = structuredClone(head);
  current.tables.users.indexes.staging_idx = {
    name: "staging_idx",
    columns: ["id"],
    isUnique: false,
  };

  assert.deepEqual(await generateEffectiveStatements(base, head, current), []);
});

test("rejects conflicting changes to the same schema value", () => {
  const base = snapshot();
  const head = structuredClone(base);
  head.tables.users.columns.id.type = "bigint";
  const current = structuredClone(base);
  current.tables.users.columns.id.type = "varchar(48)";

  assert.throws(
    () => mergeSnapshots(base, head, current),
    (error) =>
      error instanceof SnapshotMergeConflict &&
      error.message === "Conflicting schema changes at tables.users.columns.id.type",
  );
});

test("generates SQL only for the effective pull request change", async () => {
  const base = snapshot();
  const head = structuredClone(base);
  head.tables.users.columns.name = column("name", "varchar(255)");
  const current = structuredClone(base);
  current.tables.users.indexes.id_idx = {
    name: "id_idx",
    columns: ["id"],
    isUnique: false,
  };

  const statements = await generateEffectiveStatements(base, head, current);

  assert.equal(statements.length, 1);
  assert.match(statements[0], /ALTER TABLE `users` ADD `name` varchar\(255\) NOT NULL;/);
  assert.doesNotMatch(statements[0], /id_idx/);
});

test("preserves unrelated current changes when the pull request removes a column", async () => {
  const base = snapshot();
  base.tables.users.columns.old_name = column("old_name", "varchar(255)");
  const head = structuredClone(base);
  delete head.tables.users.columns.old_name;
  const current = structuredClone(base);
  current.tables.users.indexes.id_idx = {
    name: "id_idx",
    columns: ["id"],
    isUnique: false,
  };

  const statements = await generateEffectiveStatements(base, head, current);

  assert.equal(statements.length, 1);
  assert.match(statements[0], /ALTER TABLE `users` DROP COLUMN `old_name`;/);
  assert.doesNotMatch(statements[0], /id_idx/);
});
