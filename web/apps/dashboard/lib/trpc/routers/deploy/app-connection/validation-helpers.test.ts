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
vi.mock("@/lib/flags", () => ({ privateNetworking: vi.fn() }));

const scope = { appId: "caller", environmentId: "preview", targetAppId: "database" };

beforeEach(() => {
  vi.resetAllMocks();
  mocks.connections.mockResolvedValue([]);
  mocks.variables.mockResolvedValue([{ key: "DATABASE_HOST" }]);
  mocks.app.mockResolvedValue({ slug: "caller" });
});

describe("connection name availability", () => {
  it("allows default and explicit names that override user variables", async () => {
    await expect(pickDefaultName("workspace", scope, "database")).resolves.toBe("database");
    await expect(requireUnusedName("workspace", scope, "database")).resolves.toBe("database");
  });

  it("still rejects another connection's name but permits retaining the edited connection's name", async () => {
    mocks.connections.mockResolvedValue([{ id: "connection", name: "database" }]);

    await expect(pickDefaultName("workspace", scope, "database")).resolves.toBe("database-2");
    await expect(requireUnusedName("workspace", scope, "database")).rejects.toMatchObject({
      code: "CONFLICT",
    });
    await expect(requireUnusedName("workspace", scope, "database", "connection")).resolves.toBe(
      "database",
    );
  });

  it("keeps the caller's replica hostname reserved", async () => {
    await expect(pickDefaultName("workspace", scope, "caller")).resolves.toBe("caller-2");
    await expect(requireUnusedName("workspace", scope, "caller")).rejects.toMatchObject({
      code: "CONFLICT",
    });
  });
});
