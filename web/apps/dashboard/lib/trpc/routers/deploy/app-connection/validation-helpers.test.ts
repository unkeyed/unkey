import { newId } from "@unkey/id";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { pickDefaultName, requireUnusedName } from "./validation-helpers";

const mocks = vi.hoisted(() => ({
  connections: vi.fn(),
  variables: vi.fn(),
  app: vi.fn(),
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      appConnections: { findMany: mocks.connections },
      appEnvironmentVariables: { findMany: mocks.variables },
      apps: { findFirst: mocks.app },
    },
  },
}));

const workspaceId = newId("workspace");
const connectionId = newId("connection");
const scope = {
  appId: newId("app"),
  environmentId: newId("environment"),
  targetAppId: newId("app"),
};

beforeEach(() => {
  vi.resetAllMocks();
  mocks.connections.mockResolvedValue([]);
  mocks.variables.mockResolvedValue([{ key: "DATABASE_HOST" }]);
  mocks.app.mockResolvedValue({ slug: "caller" });
});

describe("connection name availability", () => {
  it("allows default and explicit names that override user variables", async () => {
    await expect(pickDefaultName(workspaceId, scope, "database")).resolves.toBe("database");
    await expect(requireUnusedName(workspaceId, scope, "database")).resolves.toBe("database");
  });

  it("still rejects another connection's name but permits retaining the edited connection's name", async () => {
    mocks.connections.mockResolvedValue([{ id: connectionId, name: "database" }]);

    await expect(pickDefaultName(workspaceId, scope, "database")).resolves.toBe("database-2");
    await expect(requireUnusedName(workspaceId, scope, "database")).rejects.toMatchObject({
      code: "CONFLICT",
    });
    await expect(requireUnusedName(workspaceId, scope, "database", connectionId)).resolves.toBe(
      "database",
    );
  });

  it("keeps the caller's replica hostname reserved", async () => {
    await expect(pickDefaultName(workspaceId, scope, "caller")).resolves.toBe("caller-2");
    await expect(requireUnusedName(workspaceId, scope, "caller")).rejects.toMatchObject({
      code: "CONFLICT",
    });
  });
});
