import { newId } from "@unkey/id";
import { describe, expect, it, vi } from "vitest";

const listApps = vi.fn();
const listHeadlines = vi.fn();
const listDisplayDomains = vi.fn();
vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ apps: { listApps } }),
  getErrorToast: vi.fn(),
}));
vi.mock("@/lib/trpc/client", () => ({
  trpcClient: {
    deploy: {
      deployment: { listHeadlines: { query: listHeadlines } },
      domain: { listDisplayDomains: { query: listDisplayDomains } },
    },
  },
}));

const { byLatestUpdate, loadProjectApps } = await import("./apps");

describe("loadProjectApps", () => {
  it("merges the SDK apps with their headline deployment and display domain", async () => {
    const projectId = newId("project");
    const deployed = { id: newId("app"), currentDeploymentId: newId("test") };
    const image = { id: newId("app"), oci: { image: "KEBAP:latest" }, updatedAt: 2 };
    const noBranch = {
      id: newId("app"),
      git: { repository: "unkeyed/unkey", defaultBranch: "" },
      updatedAt: 3,
    };
    const rows = [deployed, image, noBranch].map((app) => ({
      name: "KEBAP",
      slug: "kebap",
      sourceType: "git",
      isRolledBack: false,
      deleteProtection: false,
      createdAt: 1,
      ...app,
    }));
    listApps.mockResolvedValue({
      async *[Symbol.asyncIterator]() {
        yield { result: { data: rows } };
      },
    });
    const headline = {
      id: deployed.currentDeploymentId,
      status: "ready",
      deployedAt: 10,
      commitMessage: "ship KEBAP",
      commitSha: "abc123",
      branch: "main",
      prNumber: null,
      forkRepositoryFullName: null,
    };
    listHeadlines.mockResolvedValue([{ appId: deployed.id, ...headline }]);
    listDisplayDomains.mockResolvedValue([
      { appId: deployed.id, domain: "api.KEBAP.app", customDomain: "KEBAP.com" },
      { appId: image.id, domain: "web.KEBAP.app", customDomain: "web.KEBAP.com" },
    ]);

    const apps = await loadProjectApps(projectId, undefined);
    const byId = new Map(apps.map((app) => [app.id, app]));

    expect(apps.map((app) => app.id)).toEqual([noBranch.id, image.id, deployed.id]);

    expect(byId.get(deployed.id)).toMatchObject({
      projectId,
      headlineDeployment: headline,
      domain: "api.KEBAP.app",
      customDomain: "KEBAP.com",
    });
    // Without a current deployment the app shows no domain, even when one is assigned
    expect(byId.get(image.id)).toMatchObject({
      imageReference: "KEBAP:latest",
      headlineDeployment: null,
      domain: null,
      customDomain: null,
    });
    expect(byId.get(noBranch.id)?.defaultBranch).toBe("main");
    expect(listApps).toHaveBeenCalledWith(
      { project: projectId, limit: 100 },
      { signal: undefined },
    );
  });
});

describe("byLatestUpdate", () => {
  it("orders by updatedAt descending, never-updated last, ties by higher id", () => {
    const [low, high] = [newId("app"), newId("app")].sort();
    const apps = [
      { id: newId("app"), updatedAt: null },
      { id: newId("app"), updatedAt: 10 },
      { id: low, updatedAt: 20 },
      { id: high, updatedAt: 20 },
    ];
    expect(apps.toSorted(byLatestUpdate).map((a) => a.updatedAt)).toEqual([20, 20, 10, null]);
    expect(
      apps
        .toSorted(byLatestUpdate)
        .slice(0, 2)
        .map((a) => a.id),
    ).toEqual([high, low]);
  });
});
