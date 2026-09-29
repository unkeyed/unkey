import { beforeEach, describe, expect, it, vi } from "vitest";
import { pickDefaultName, requireUnusedName } from "./validation-helpers";

const mocks = vi.hoisted(() => ({
  bindings: vi.fn(),
  variables: vi.fn(),
  app: vi.fn(),
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      appBindings: { findMany: mocks.bindings },
      appEnvironmentVariables: { findMany: mocks.variables },
      apps: { findFirst: mocks.app },
    },
  },
}));
vi.mock("@/lib/flags", () => ({ privateNetworking: vi.fn() }));

const scope = { appId: "caller", environmentId: "preview", targetAppId: "database" };

beforeEach(() => {
  vi.resetAllMocks();
  mocks.bindings.mockResolvedValue([]);
  mocks.variables.mockResolvedValue([{ key: "DATABASE_HOST" }]);
  mocks.app.mockResolvedValue({ slug: "caller" });
});

describe("binding name availability", () => {
  it("allows default and explicit names that override user variables", async () => {
    await expect(pickDefaultName("workspace", scope, "database")).resolves.toBe("database");
    await expect(requireUnusedName("workspace", scope, "database")).resolves.toBe("database");
  });

  it("still rejects another binding's name but permits retaining the edited binding's name", async () => {
    mocks.bindings.mockResolvedValue([{ id: "binding", name: "database" }]);

    await expect(pickDefaultName("workspace", scope, "database")).resolves.toBe("database-2");
    await expect(requireUnusedName("workspace", scope, "database")).rejects.toMatchObject({
      code: "CONFLICT",
    });
    await expect(requireUnusedName("workspace", scope, "database", "binding")).resolves.toBe(
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
