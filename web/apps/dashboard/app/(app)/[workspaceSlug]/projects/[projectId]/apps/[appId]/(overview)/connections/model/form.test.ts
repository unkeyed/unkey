import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import type { Environment, ListedConnection } from "../connection-rules";
import {
  type ConnectionFormValues,
  IMAGE_PREVIEW_ERROR,
  connectionFormSchema,
  defaultEnvs,
  formValuesOf,
  planRemove,
  planSave,
} from "./form";
import type { ConnectionEnvs, MergedConnection } from "./merge";

const appId = newId("app");
const db = newId("app");
const production: Environment = {
  id: newId("environment"),
  appId,
  slug: "production",
  kind: "production",
};
const preview: Environment = { id: newId("environment"), appId, slug: "preview", kind: "preview" };
const envs: ConnectionEnvs = { production, preview };
const dbStaging = newId("environment");

function listed(environment: Environment, overrides: Partial<ListedConnection> = {}) {
  return {
    id: newId("connection"),
    appId,
    environmentId: environment.id,
    targetAppId: db,
    targetAppName: "db",
    name: "db",
    targetType: "automatic",
    targetAppSlug: "db",
    ...overrides,
  } satisfies ListedConnection;
}

function connected(byEnv: MergedConnection["byEnv"]): MergedConnection {
  return { targetAppId: db, targetAppName: "db", name: "db", byEnv };
}

describe("planSave", () => {
  it("creates one connection per enabled environment on add", () => {
    const values: ConnectionFormValues = { targetAppId: db, name: "", envs: defaultEnvs(null) };

    expect(planSave(null, values, envs)).toEqual([
      {
        type: "create",
        env: "production",
        environmentId: production.id,
        rule: { targetType: "automatic" },
      },
      {
        type: "create",
        env: "preview",
        environmentId: preview.id,
        rule: { targetType: "automatic" },
      },
    ]);
  });

  it("passes a chosen name and skips disabled environments on add", () => {
    const values: ConnectionFormValues = {
      targetAppId: db,
      name: " primary ",
      envs: {
        ...defaultEnvs(dbStaging),
        production: { enabled: false, target: { mode: "automatic" } },
      },
    };

    expect(planSave(null, values, envs)).toEqual([
      {
        type: "create",
        env: "preview",
        environmentId: preview.id,
        rule: { targetType: "environment", targetEnvironmentId: dbStaging },
        name: "primary",
      },
    ]);
  });

  it("emits nothing when the edit form is unchanged", () => {
    const before = connected({ production: listed(production), preview: listed(preview) });

    expect(planSave(before, formValuesOf(before, defaultEnvs(null)), envs)).toEqual([]);
  });

  it("updates, creates and deletes only the environments that changed", () => {
    const productionRow = listed(production);
    const before = connected({ production: productionRow });
    const values: ConnectionFormValues = {
      targetAppId: db,
      name: "db",
      envs: {
        production: { enabled: true, target: { mode: "environment", environmentId: dbStaging } },
        preview: { enabled: true, target: { mode: "automatic" } },
      },
    };

    expect(planSave(before, values, envs)).toEqual([
      {
        type: "update",
        env: "production",
        id: productionRow.id,
        rule: { targetType: "environment", targetEnvironmentId: dbStaging },
      },
      {
        type: "create",
        env: "preview",
        environmentId: preview.id,
        rule: { targetType: "automatic" },
        name: "db",
      },
    ]);

    const off = {
      ...values,
      envs: {
        ...values.envs,
        production: { enabled: false, target: { mode: "automatic" as const } },
      },
    };
    expect(planSave(before, off, envs)).toEqual([
      { type: "delete", env: "production", id: productionRow.id },
      {
        type: "create",
        env: "preview",
        environmentId: preview.id,
        rule: { targetType: "automatic" },
        name: "db",
      },
    ]);
  });

  it("renames every environment that does not already use the new name", () => {
    const productionRow = listed(production);
    const previewRow = listed(preview, { name: "primary" });
    const before = connected({ production: productionRow, preview: previewRow });
    const values = { ...formValuesOf(before, defaultEnvs(null)), name: "primary" };

    expect(planSave(before, values, envs)).toEqual([
      {
        type: "update",
        env: "production",
        id: productionRow.id,
        rule: { targetType: "automatic" },
        name: "primary",
      },
    ]);
  });
});

describe("planRemove", () => {
  it("deletes the connection in every environment", () => {
    const productionRow = listed(production);
    const previewRow = listed(preview);

    expect(planRemove(connected({ production: productionRow, preview: previewRow }))).toEqual([
      { type: "delete", env: "production", id: productionRow.id },
      { type: "delete", env: "preview", id: previewRow.id },
    ]);
  });
});

describe("connectionFormSchema", () => {
  const values: ConnectionFormValues = { targetAppId: db, name: "", envs: defaultEnvs(null) };
  const issues = (schema: ReturnType<typeof connectionFormSchema>, input: ConnectionFormValues) =>
    schema.safeParse(input).error?.issues.map((issue) => [issue.path.join("."), issue.message]);

  it("lets the server pick the name on add but requires one on edit", () => {
    expect(
      issues(
        connectionFormSchema({ requiresExplicitPreviewTarget: false, nameRequired: false }),
        values,
      ),
    ).toBeUndefined();
    expect(
      issues(
        connectionFormSchema({ requiresExplicitPreviewTarget: false, nameRequired: true }),
        values,
      ),
    ).toEqual([["name", "Name is required"]]);
  });

  it("requires an app, an environment and a complete target", () => {
    const schema = connectionFormSchema({
      requiresExplicitPreviewTarget: false,
      nameRequired: false,
    });

    expect(
      issues(schema, {
        targetAppId: "",
        name: "",
        envs: {
          production: { enabled: false, target: { mode: "automatic" } },
          preview: { enabled: false, target: { mode: "deployment", deploymentId: "" } },
        },
      }),
    ).toEqual([
      ["targetAppId", "Choose an app"],
      ["envs", "Turn on at least one environment"],
    ]);
    expect(
      issues(schema, {
        ...values,
        envs: {
          production: { enabled: true, target: { mode: "environment", environmentId: "" } },
          preview: { enabled: true, target: { mode: "deployment", deploymentId: "" } },
        },
      }),
    ).toEqual([
      ["envs.production.target", "Choose an environment"],
      ["envs.preview.target", "Choose a deployment"],
    ]);
  });

  it("asks image apps for an explicit preview target", () => {
    expect(
      issues(
        connectionFormSchema({ requiresExplicitPreviewTarget: true, nameRequired: false }),
        values,
      ),
    ).toEqual([["envs.preview.target", IMAGE_PREVIEW_ERROR]]);
  });
});
