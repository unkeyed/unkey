import { createRequire } from "node:module";
import { sql } from "drizzle-orm";
import {
  boolean,
  index,
  int,
  mysqlTable,
  unique,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { describe, expect, it } from "vitest";
import { planBranchChanges } from "./pscale-branch.mjs";

const { generateMySQLDrizzleJson: snapshot }: typeof import("drizzle-kit/api") = createRequire(
  import.meta.url,
)("drizzle-kit/api");

describe("PlanetScale branch schema changes", () => {
  it("applies only the Git delta, preserving newer tables, columns, indexes and definitions", async () => {
    const base = await snapshot({
      apps: mysqlTable("apps", { name: varchar("name", { length: 32 }) }),
      deployments: mysqlTable("deployments", {
        attempts: int("attempts").default(1),
        retired: int("retired"),
      }),
    });
    const head = await snapshot({
      apps: mysqlTable("apps", { name: varchar("name", { length: 32 }) }),
      deployments: mysqlTable("deployments", {
        attempts: int("attempts").default(1),
        retired: int("retired"),
        region: varchar("region", { length: 64 }),
      }),
    });
    const live = await snapshot({
      apps: mysqlTable("apps", { name: varchar("name", { length: 256 }) }),
      deployments: mysqlTable(
        "deployments",
        { attempts: int("attempts").default(7), newer: int("newer") },
        (table) => [index("newer_idx").on(table.newer)],
      ),
      projects: mysqlTable("projects", { name: varchar("name", { length: 80 }) }),
    });

    expect(await planBranchChanges(base, head, live)).toEqual([
      "ALTER TABLE `deployments` ADD `region` varchar(64);",
    ]);
    expect(await planBranchChanges(base, base, live)).toEqual([]);
  });

  it("supports additions, changes and removals, and does nothing on an identical rerun", async () => {
    const base = await snapshot({
      deployments: mysqlTable("deployments", {
        attempts: int("attempts").default(1),
        retired: int("retired"),
      }),
    });
    const head = await snapshot({
      deployments: mysqlTable("deployments", { attempts: int("attempts").default(3) }),
      regions: mysqlTable("regions", { name: varchar("name", { length: 32 }) }),
    });
    const sql = await planBranchChanges(base, head, base);
    expect(sql).toContain("ALTER TABLE `deployments` MODIFY COLUMN `attempts` int DEFAULT 3;");
    expect(sql).toContain("ALTER TABLE `deployments` DROP COLUMN `retired`;");
    expect(sql.some((statement) => statement.startsWith("CREATE TABLE `regions`"))).toBe(true);
    expect(await planBranchChanges(base, head, head)).toEqual([]);
  });

  it("recognizes MySQL's boolean and unique-index representations on reruns", async () => {
    const base = await snapshot({});
    const head = await snapshot({
      apps: mysqlTable("apps", { enabled: boolean("enabled").default(false) }, (table) => [
        uniqueIndex("enabled_idx").on(table.enabled),
      ]),
    });
    const live = await snapshot({
      apps: mysqlTable("apps", { enabled: boolean("enabled").default(false) }, (table) => [
        unique("enabled_idx").on(table.enabled),
      ]),
    });
    live.tables.apps.columns.enabled.type = "tinyint(1)";
    live.tables.apps.columns.enabled.default = 0;
    expect(await planBranchChanges(base, head, live)).toEqual([]);
  });

  it("keeps expression-index metadata when applying a new index", async () => {
    const base = await snapshot({
      apps: mysqlTable("apps", { name: varchar("name", { length: 32 }) }),
    });
    const head = await snapshot({
      apps: mysqlTable("apps", { name: varchar("name", { length: 32 }) }, (table) => [
        index("lower_name_idx").on(sql`(lower(${table.name}))`),
      ]),
    });
    expect(await planBranchChanges(base, head, base)).toEqual([
      "CREATE INDEX `lower_name_idx` ON `apps` ((lower(`name`)));",
    ]);
  });

  it("rejects conflicting changes to the same object before returning SQL", async () => {
    const base = await snapshot({
      deployments: mysqlTable("deployments", { attempts: int("attempts").default(1) }),
    });
    const head = await snapshot({
      deployments: mysqlTable("deployments", { attempts: int("attempts").default(3) }),
    });
    const live = await snapshot({
      deployments: mysqlTable("deployments", { attempts: int("attempts").default(7) }),
    });
    await expect(planBranchChanges(base, head, live)).rejects.toThrow(
      "Schema conflict at deployments.columns.attempts",
    );
  });

  it("rejects column and table rename ambiguity without opening an input prompt", async () => {
    const base = await snapshot({
      apps: mysqlTable("apps", { oldName: int("old_name") }),
    });
    const renamedColumn = await snapshot({
      apps: mysqlTable("apps", { newName: int("new_name") }),
    });
    const renamedTable = await snapshot({
      projects: mysqlTable("projects", { oldName: int("old_name") }),
    });
    await expect(planBranchChanges(base, renamedColumn, base)).rejects.toThrow(
      "Possible rename in apps.columns",
    );
    await expect(planBranchChanges(base, renamedTable, base)).rejects.toThrow(
      "Possible rename in tables",
    );
  });
});
