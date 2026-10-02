import { createLiveQueryCollection, eq } from "@tanstack/react-db";
import { describe, expect, it, vi } from "vitest";

const appRows = [
  {
    id: "app_a",
    name: "api",
    slug: "api",
    sourceType: "git",
    currentDeploymentId: "dep_a",
    isRolledBack: false,
    deleteProtection: false,
    createdAt: 1,
    updatedAt: 100,
  },
  {
    id: "app_b",
    name: "web",
    slug: "web",
    sourceType: "oci",
    oci: { image: "KEBAP:latest" },
    isRolledBack: false,
    deleteProtection: false,
    createdAt: 2,
  },
  {
    id: "app_c",
    name: "worker",
    slug: "worker",
    sourceType: "git",
    git: { repository: "unkeyed/unkey", defaultBranch: "" },
    isRolledBack: false,
    deleteProtection: false,
    createdAt: 3,
    updatedAt: 200,
  },
];
const headlineA = {
  id: "dep_a",
  status: "ready",
  deployedAt: 10,
  commitMessage: "ship KEBAP",
  commitSha: "abc123",
  branch: "main",
  prNumber: null,
  forkRepositoryFullName: null,
};
const headlineB = { ...headlineA, id: "dep_b", status: "failed", deployedAt: 20 };

const listApps = vi.fn(async (_input: { project: string; limit: number }) => ({
  result: { data: appRows },
  async *[Symbol.asyncIterator]() {
    yield { result: { data: appRows } };
  },
}));
const listHeadlines = vi.fn(async (_input: { projectId: string }) => [
  { appId: "app_a", ...headlineA },
  { appId: "app_b", ...headlineB },
]);
const listDisplayDomains = vi.fn(async (_input: { projectId: string }) => [
  { appId: "app_a", domain: "api.KEBAP.app", customDomain: "KEBAP.com" },
  { appId: "app_b", domain: "web.KEBAP.app", customDomain: "web.KEBAP.com" },
]);

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ apps: { listApps } }),
  getErrorToast: vi.fn(),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: vi.fn() } }));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return {
    queryClient: new QueryClient(),
    trpcClient: {
      deploy: {
        deployment: { listHeadlines: { query: listHeadlines } },
        domain: { listDisplayDomains: { query: listDisplayDomains } },
      },
    },
  };
});

const { apps } = await import("./apps");

describe("apps collection", () => {
  it("merges the SDK apps with their headline deployment and display domain", async () => {
    const query = createLiveQueryCollection((q) =>
      q
        .from({ app: apps })
        .where(({ app }) => eq(app.projectId, "proj_1"))
        .orderBy(({ app }) => app.updatedAt, { direction: "desc", nulls: "last" })
        .orderBy(({ app }) => app.id, "desc"),
    );
    await query.preload();

    await vi.waitFor(() =>
      expect(query.toArray.map((app) => app.id)).toEqual(["app_c", "app_a", "app_b"]),
    );
    const byId = new Map(query.toArray.map((app) => [app.id, app]));

    expect(byId.get("app_a")?.headlineDeployment).toEqual(headlineA);
    expect(byId.get("app_a")?.domain).toBe("api.KEBAP.app");
    expect(byId.get("app_a")?.customDomain).toBe("KEBAP.com");
    expect(byId.get("app_b")?.headlineDeployment).toEqual(headlineB);
    expect(byId.get("app_b")?.domain).toBeNull();
    expect(byId.get("app_b")?.customDomain).toBeNull();
    expect(byId.get("app_b")?.imageReference).toBe("KEBAP:latest");
    expect(byId.get("app_c")?.headlineDeployment).toBeNull();
    expect(byId.get("app_c")?.defaultBranch).toBe("main");

    expect(listApps).toHaveBeenCalledWith({ project: "proj_1", limit: 100 });
    expect(listHeadlines).toHaveBeenCalledWith({ projectId: "proj_1" });
    expect(listDisplayDomains).toHaveBeenCalledWith({ projectId: "proj_1" });
  });
});
