import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import type { Environment, ListedConnection } from "../connection-rules";
import { type ConnectionEnvs, mergeConnections } from "./merge";

const appId = newId("app");
const production: Environment = {
  id: newId("environment"),
  appId,
  slug: "production",
  kind: "production",
};
const preview: Environment = { id: newId("environment"), appId, slug: "preview", kind: "preview" };
const envs: ConnectionEnvs = { production, preview };
const api = newId("app");
const db = newId("app");

function row(environment: Environment, targetAppId: string, name: string): ListedConnection {
  return {
    id: newId("connection"),
    appId,
    environmentId: environment.id,
    targetAppId,
    targetAppName: name,
    name,
    targetType: "automatic",
    targetAppSlug: name,
  };
}

describe("mergeConnections", () => {
  it("pairs each target app's rows across environments, sorted by app name", () => {
    const dbPreview = row(preview, db, "db");
    const apiProduction = row(production, api, "api");
    const dbProduction = row(production, db, "db");

    expect(mergeConnections([dbPreview, apiProduction, dbProduction], envs)).toEqual([
      {
        targetAppId: api,
        targetAppName: "api",
        name: "api",
        byEnv: { production: apiProduction },
      },
      {
        targetAppId: db,
        targetAppName: "db",
        name: "db",
        byEnv: { production: dbProduction, preview: dbPreview },
      },
    ]);
  });

  it("names the connection after production, else preview", () => {
    const previewRow = { ...row(preview, db, "db"), name: "database" };
    const productionRow = { ...row(production, db, "db"), name: "primary" };

    expect(mergeConnections([previewRow], envs)[0]?.name).toBe("database");
    expect(mergeConnections([previewRow, productionRow], envs)[0]?.name).toBe("primary");
  });

  it("ignores rows from environments outside the pair", () => {
    const canary: Environment = {
      id: newId("environment"),
      appId,
      slug: "canary",
      kind: "production",
    };

    expect(mergeConnections([row(canary, db, "db")], envs)).toEqual([]);
    expect(mergeConnections([row(preview, db, "db")], { production, preview: null })).toEqual([]);
  });
});
