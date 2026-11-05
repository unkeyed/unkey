import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import type { CallerEnvironment, Callers } from "./callers";
import {
  type ConnectionFormValues,
  connectionFormSchema,
  openForm,
  takenNames,
  targetsForApp,
} from "./form";
import { type SaveIntent, planSave } from "./plan";
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

describe("targetsForApp", () => {
  const orphan = listed({
    environmentId: previewEnv.id,
    targetType: "environment",
    targetEnvironmentId: dbStaging,
  });

  it("seeds the preview target from a preview-only row", () => {
    const current = { ...values, targetAppId: "", targets: { production: staging, preview: null } };
    expect(targetsForApp(callersOf([], [orphan]), current, db)).toEqual({
      production: automatic,
      preview: staging,
    });
  });

  it("resets connected targets to automatic otherwise", () => {
    const current = (preview: TargetRule | null) => ({
      ...values,
      targetAppId: "",
      targets: { production: staging, preview },
    });
    expect(targetsForApp(callersOf([], []), current(staging), db)).toEqual({
      production: automatic,
      preview: automatic,
    });
    expect(targetsForApp(callersOf([], []), current(null), db)).toEqual({
      production: automatic,
      preview: null,
    });
  });

  it("does not carry a preview seeded from the previous app's row", () => {
    const api = newId("app");
    expect(targetsForApp(callersOf([], [orphan]), withPreview(staging), api)).toEqual({
      production: automatic,
      preview: null,
    });
  });
});

describe("openForm", () => {
  it("reads the app, name and both targets of a row", () => {
    expect(
      openForm(
        listed({ targetType: "deployment", targetDeploymentId: undefined }),
        callersOf([], null),
      ).values,
    ).toEqual({
      targetAppId: db,
      name: "db",
      targets: { production: { targetType: "deployment", targetDeploymentId: "" }, preview: null },
    });
    const row = listed();
    const companion = listed({
      environmentId: previewEnv.id,
      targetType: "environment",
      targetEnvironmentId: dbStaging,
    });
    expect(openForm(row, callersOf([], [companion]))).toEqual({
      values: withPreview(staging),
      intent: {
        kind: "edit",
        opened: { id: row.id, name: "db", production: automatic, preview: staging },
      },
    });
  });

  it("starts empty when adding", () => {
    expect(openForm(null, callersOf([], []))).toEqual({
      values: { targetAppId: "", name: "", targets: { production: automatic, preview: null } },
      intent: { kind: "add" },
    });
  });
});

describe("takenNames", () => {
  const taken = (callers: Callers, input: ConnectionFormValues, intent: SaveIntent) =>
    takenNames(callers, input.targetAppId, planSave(callers, input, intent));

  it("counts preview names only while previews are connected", () => {
    const callers = callersOf(
      [listed({ targetAppId: newId("app"), name: "api" })],
      [listed({ targetAppId: newId("app"), name: "cache" })],
    );
    expect(taken(callers, values, ADD)).toEqual(new Set(["api"]));
    expect(taken(callers, withPreview(automatic), ADD)).toEqual(new Set(["api", "cache"]));
  });

  it("counts preview names only when the preview save sets a name", () => {
    const row = listed();
    const callers = callersOf(
      [row, listed({ targetAppId: newId("app"), name: "api" })],
      [
        listed({ environmentId: previewEnv.id }),
        listed({ targetAppId: newId("app"), name: "cache" }),
      ],
    );
    const intent: SaveIntent = {
      kind: "edit",
      opened: { id: row.id, name: "db", production: automatic, preview: automatic },
    };
    expect(taken(callers, withPreview(staging), intent)).toEqual(new Set(["api"]));
    expect(taken(callers, withPreview(staging, "database"), intent)).toEqual(
      new Set(["api", "cache"]),
    );
  });

  it("always counts production names, even with no planned ops", () => {
    const callers = callersOf(
      [listed({ targetAppId: newId("app"), name: "api" })],
      [listed({ targetAppId: newId("app"), name: "cache" })],
    );
    expect(takenNames(callers, db, [])).toEqual(new Set(["api"]));
  });
});

describe("connectionFormSchema", () => {
  const issues = (schema: ReturnType<typeof connectionFormSchema>, input: ConnectionFormValues) =>
    schema.safeParse(input).error?.issues.map((issue) => [issue.path.join("."), issue.message]);

  it("rejects a name another connection in the environment uses", () => {
    const api = newId("app");
    const schema = connectionFormSchema(
      callersOf([listed({ targetAppId: api, name: "api" }), listed()], []),
      ADD,
    );

    expect(issues(schema, { ...values, name: "api" })).toEqual([
      ["name", "Another connection already uses the name api."],
    ]);
    expect(issues(schema, values)).toBeUndefined();
  });

  it("checks preview names only while previews are connected", () => {
    const schema = connectionFormSchema(
      callersOf([], [listed({ targetAppId: newId("app"), name: "api" })]),
      ADD,
    );
    expect(issues(schema, { ...values, name: "api" })).toBeUndefined();
    expect(issues(schema, withPreview(automatic, "api"))).toEqual([
      ["name", "Another connection already uses the name api."],
    ]);
  });

  it("requires a valid name", () => {
    const schema = connectionFormSchema(callersOf([], []), ADD);
    expect(issues(schema, { ...values, name: " " })).toEqual([["name", "Name is required"]]);
  });

  it("requires an app and complete targets", () => {
    const schema = connectionFormSchema(callersOf([], []), ADD);
    expect(
      issues(schema, {
        targetAppId: "",
        name: "db",
        targets: {
          production: { targetType: "deployment", targetDeploymentId: "" },
          preview: { targetType: "environment", targetEnvironmentId: "" },
        },
      }),
    ).toEqual([
      ["targetAppId", "Choose an app"],
      ["targets.production", "Choose a deployment"],
      ["targets.preview", "Choose an environment"],
    ]);
  });
});
