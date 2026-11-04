import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import { type CallerEnvironment, connectedAppIds } from "./callers";
import type { ListedConnection } from "./target";

describe("connectedAppIds", () => {
  it("includes only apps connected in production", () => {
    const production: CallerEnvironment = { id: newId("environment"), kind: "production" };
    const preview: CallerEnvironment = { id: newId("environment"), kind: "preview" };
    const row = (targetAppId: string, environmentId: string): ListedConnection => ({
      id: newId("connection"),
      appId: newId("app"),
      environmentId,
      targetAppId,
      targetAppName: "db",
      name: "db",
      targetType: "automatic",
      targetAppSlug: "db",
    });
    const db = newId("app");
    const api = newId("app");
    expect(
      connectedAppIds({
        production: { environment: production, connections: [row(db, production.id)] },
        preview: { environment: preview, connections: [row(api, preview.id)] },
      }),
    ).toEqual(new Set([db]));
  });
});
