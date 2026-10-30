import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import { type ConnectionFormValues, connectionFormSchema, formValuesOf, planSave } from "./form";
import type { ListedConnection, TargetRule } from "./target";

const appId = newId("app");
const db = newId("app");
const environmentId = newId("environment");
const dbStaging = newId("environment");

function listed(overrides: Partial<ListedConnection> = {}) {
  return {
    id: newId("connection"),
    appId,
    environmentId,
    targetAppId: db,
    targetAppName: "db",
    name: "db",
    targetType: "automatic",
    targetAppSlug: "db",
    ...overrides,
  } satisfies ListedConnection;
}

const values: ConnectionFormValues = {
  targetAppId: db,
  name: "db",
  target: { targetType: "automatic" },
};

describe("planSave", () => {
  it("creates the connection when adding", () => {
    expect(planSave(null, { ...values, name: " db " })).toEqual({
      type: "create",
      targetAppId: db,
      rule: { targetType: "automatic" },
      name: "db",
    });
  });

  it("emits nothing when the edited row is unchanged", () => {
    expect(planSave(listed(), values)).toBeNull();
  });

  it("updates the edited row, never creates, and only sends the name when it changed", () => {
    const row = listed();
    const staging: TargetRule = { targetType: "environment", targetEnvironmentId: dbStaging };
    expect(planSave(row, { ...values, target: staging })).toEqual({
      type: "update",
      id: row.id,
      rule: staging,
    });
    expect(planSave(row, { ...values, name: "database" })).toEqual({
      type: "update",
      id: row.id,
      rule: { targetType: "automatic" },
      name: "database",
    });
  });
});

describe("formValuesOf", () => {
  it("reads the app, name and target of a row", () => {
    expect(
      formValuesOf(listed({ targetType: "deployment", targetDeploymentId: undefined })),
    ).toEqual({
      targetAppId: db,
      name: "db",
      target: { targetType: "deployment", targetDeploymentId: "" },
    });
  });
});

describe("connectionFormSchema", () => {
  const issues = (schema: ReturnType<typeof connectionFormSchema>, input: ConnectionFormValues) =>
    schema.safeParse(input).error?.issues.map((issue) => [issue.path.join("."), issue.message]);

  it("rejects a name another connection in the environment uses", () => {
    const api = newId("app");
    const schema = connectionFormSchema([listed({ targetAppId: api, name: "api" }), listed()]);

    expect(issues(schema, { ...values, name: "api" })).toEqual([
      ["name", "Another connection already uses the name api."],
    ]);
    expect(issues(schema, values)).toBeUndefined();
  });

  it("requires a valid name", () => {
    const schema = connectionFormSchema([]);
    expect(issues(schema, { ...values, name: " " })).toEqual([["name", "Name is required"]]);
  });

  it("requires an app and a complete target", () => {
    const schema = connectionFormSchema([]);
    expect(
      issues(schema, {
        targetAppId: "",
        name: "db",
        target: { targetType: "deployment", targetDeploymentId: "" },
      }),
    ).toEqual([
      ["targetAppId", "Choose an app"],
      ["target", "Choose a deployment"],
    ]);
  });
});
