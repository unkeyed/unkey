import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import type { CallerEnvironment, Callers } from "./callers";
import type { ConnectionFormValues } from "./form";
import { type SaveIntent, planSave, removalOf, removesPreview, renameOf } from "./plan";
import type { ListedConnection, TargetRule } from "./target";

const appId = newId("app");
const db = newId("app");
const production: CallerEnvironment = { id: newId("environment"), kind: "production" };
const previewEnv: CallerEnvironment = { id: newId("environment"), kind: "preview" };
const dbStaging = newId("environment");

function listed(overrides: Partial<ListedConnection> = {}) {
  return {
    id: newId("connection"),
    appId,
    environmentId: production.id,
    targetAppId: db,
    targetAppName: "db",
    name: "db",
    targetType: "automatic",
    targetAppSlug: "db",
    ...overrides,
  } satisfies ListedConnection;
}

const automatic: TargetRule = { targetType: "automatic" };
const staging: TargetRule = { targetType: "environment", targetEnvironmentId: dbStaging };

const values: ConnectionFormValues = {
  targetAppId: db,
  name: "db",
  targets: { production: automatic, preview: null },
};

const withPreview = (preview: TargetRule | null, name = "db"): ConnectionFormValues => ({
  ...values,
  name,
  targets: { production: automatic, preview },
});

function callersOf(
  productionRows: ListedConnection[],
  previewRows: ListedConnection[] | null,
): Callers {
  return {
    production: { environment: production, connections: productionRows },
    preview: previewRows ? { environment: previewEnv, connections: previewRows } : null,
  };
}

const ADD: SaveIntent = { kind: "add" };

const editOf = (
  row: ListedConnection,
  preview: TargetRule | null = null,
  production: TargetRule = automatic,
): SaveIntent => ({
  kind: "edit",
  opened: { id: row.id, name: row.name, production, preview },
});

describe("planSave", () => {
  it("creates the production connection when adding", () => {
    expect(planSave(callersOf([], null), { ...values, name: " db " }, ADD)).toEqual([
      { type: "create", environment: production, targetAppId: db, rule: automatic, name: "db" },
    ]);
  });

  it("creates production first, then the preview connection", () => {
    expect(planSave(callersOf([], []), withPreview(automatic), ADD)).toEqual([
      { type: "create", environment: production, targetAppId: db, rule: automatic, name: "db" },
      { type: "create", environment: previewEnv, targetAppId: db, rule: automatic, name: "db" },
    ]);
  });

  it("creates nothing in preview while previews are not connected", () => {
    expect(planSave(callersOf([], []), values, ADD).map((op) => op.environment)).toEqual([
      production,
    ]);
  });

  it("creates only the missing preview row when a partial add is retried", () => {
    expect(planSave(callersOf([listed()], []), withPreview(automatic), ADD)).toEqual([
      { type: "create", environment: previewEnv, targetAppId: db, rule: automatic, name: "db" },
    ]);
  });

  it("adopts an existing preview-only row instead of creating a second one", () => {
    const orphan = listed({ environmentId: previewEnv.id, name: "old-db" });
    expect(planSave(callersOf([], [orphan]), withPreview(staging), ADD)).toEqual([
      { type: "create", environment: production, targetAppId: db, rule: automatic, name: "db" },
      { type: "update", environment: previewEnv, id: orphan.id, rule: staging, name: "db" },
    ]);
  });

  it("emits nothing when neither connection changed", () => {
    const row = listed();
    const companion = listed({ environmentId: previewEnv.id });
    expect(
      planSave(callersOf([row], [companion]), withPreview(automatic), editOf(row, automatic)),
    ).toEqual([]);
    expect(planSave(callersOf([row], []), values, editOf(row))).toEqual([]);
  });

  it("updates only the connection whose target changed", () => {
    const row = listed();
    const companion = listed({ environmentId: previewEnv.id });
    const callers = callersOf([row], [companion]);
    const intent = editOf(row, automatic);
    expect(planSave(callers, withPreview(staging), intent)).toEqual([
      { type: "update", environment: previewEnv, id: companion.id, rule: staging },
    ]);
    expect(
      planSave(
        callers,
        { ...values, targets: { production: staging, preview: automatic } },
        intent,
      ),
    ).toEqual([{ type: "update", environment: production, id: row.id, rule: staging }]);
  });

  it("applies a rename to both connections", () => {
    const row = listed();
    const companion = listed({ environmentId: previewEnv.id });
    expect(
      planSave(
        callersOf([row], [companion]),
        withPreview(automatic, "database"),
        editOf(row, automatic),
      ),
    ).toEqual([
      { type: "update", environment: production, id: row.id, rule: automatic, name: "database" },
      {
        type: "update",
        environment: previewEnv,
        id: companion.id,
        rule: automatic,
        name: "database",
      },
    ]);
  });

  it("leaves a differently named companion alone unless production is renamed", () => {
    const row = listed();
    const companion = listed({ environmentId: previewEnv.id, name: "legacy-db" });
    const callers = callersOf([row], [companion]);
    expect(planSave(callers, withPreview(automatic), editOf(row, automatic))).toEqual([]);
    expect(planSave(callers, withPreview(staging), editOf(row, automatic))).toEqual([
      { type: "update", environment: previewEnv, id: companion.id, rule: staging },
    ]);
  });

  it("renames preview companion when a partial rename is retried", () => {
    const opened = listed();
    const renamed = { ...opened, name: "database" };
    const companion = listed({ environmentId: previewEnv.id });
    expect(
      planSave(
        callersOf([renamed], [companion]),
        withPreview(automatic, "database"),
        editOf(opened, automatic),
      ),
    ).toEqual([
      {
        type: "update",
        environment: previewEnv,
        id: companion.id,
        rule: automatic,
        name: "database",
      },
    ]);
  });

  it("updates the opened connection when its production row is gone", () => {
    const opened = listed();
    expect(planSave(callersOf([], []), withPreview(null, "database"), editOf(opened))).toEqual([
      { type: "update", environment: production, id: opened.id, rule: automatic, name: "database" },
    ]);
  });

  it("updates the opened connection when its production row is gone and only preview changed", () => {
    const opened = listed();
    const companion = listed({ environmentId: previewEnv.id });
    expect(
      planSave(callersOf([], [companion]), withPreview(staging), editOf(opened, automatic)),
    ).toEqual([
      { type: "update", environment: production, id: opened.id, rule: automatic },
      { type: "update", environment: previewEnv, id: companion.id, rule: staging },
    ]);
    expect(
      planSave(callersOf([], [companion]), withPreview(automatic), editOf(opened, automatic)),
    ).toEqual([]);
  });

  it("creates the preview companion when editing a production-only connection", () => {
    const row = listed();
    expect(planSave(callersOf([row], []), withPreview(automatic), editOf(row))).toEqual([
      { type: "create", environment: previewEnv, targetAppId: db, rule: automatic, name: "db" },
    ]);
  });

  it("deletes the preview companion when previews are disconnected", () => {
    const row = listed();
    const companion = listed({ environmentId: previewEnv.id });
    expect(planSave(callersOf([row], [companion]), values, editOf(row, automatic))).toEqual([
      { type: "delete", environment: previewEnv, id: companion.id },
    ]);
  });

  it("deletes preview only when it was connected at open", () => {
    const row = listed();
    const companion = listed({ environmentId: previewEnv.id });
    expect(planSave(callersOf([row], [companion]), values, editOf(row))).toEqual([]);
    expect(planSave(callersOf([], [companion]), values, ADD)).toEqual([
      { type: "create", environment: production, targetAppId: db, rule: automatic, name: "db" },
    ]);
  });
});

describe("removalOf", () => {
  it("deletes the preview companion before the production connection", () => {
    const row = listed();
    const companion = listed({ environmentId: previewEnv.id });
    const other = listed({ targetAppId: newId("app"), environmentId: previewEnv.id });
    expect(removalOf(callersOf([row], [other, companion]), db)).toEqual([
      { type: "delete", environment: previewEnv, id: companion.id },
      { type: "delete", environment: production, id: row.id },
    ]);
    expect(removalOf(callersOf([row], [other]), db)).toEqual([
      { type: "delete", environment: production, id: row.id },
    ]);
  });
});

describe("removesPreview", () => {
  it("is true only when a preview row is deleted", () => {
    const row = listed();
    const companion = listed({ environmentId: previewEnv.id });
    expect(removesPreview(removalOf(callersOf([row], [companion]), db))).toBe(true);
    expect(removesPreview(removalOf(callersOf([row], []), db))).toBe(false);
    expect(removesPreview(removalOf(callersOf([row], null), db))).toBe(false);
  });
});

describe("renameOf", () => {
  it("reads the new name against the name at open", () => {
    const opened = listed();
    expect(renameOf(opened, " database ")).toBe("database");
    expect(renameOf(opened, "db")).toBeNull();
    expect(renameOf(opened, " ")).toBeNull();
    expect(renameOf(null, "database")).toBeNull();
  });
});
