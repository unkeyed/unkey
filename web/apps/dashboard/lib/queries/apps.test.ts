import { newId } from "@unkey/id";
import { describe, expect, it, vi } from "vitest";

const listApps = vi.fn();
const listHeadlines = vi.fn();
const listDisplayDomains = vi.fn();
vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ apps: { listApps } }),
  getErrorToast: vi.fn(),
}));
vi.mock("@/lib/collections/client", () => ({
  trpcClient: {
    deploy: {
      deployment: { listHeadlines: { query: listHeadlines } },
      domain: { listDisplayDomains: { query: listDisplayDomains } },
    },
  },
}));

const { loadProjectApps } = await import("./apps");

describe("loadProjectApps", () => {
  it("merges the SDK apps with their headline deployment and display domain", async () => {
    const projectId = newId("project");
    const deployed = { id: newId("app"), currentDeploymentId: newId("test") };
    const image = { id: newId("app"), oci: { image: "KEBAP:latest" } };
    const noBranch = { id: newId("app"), git: { repository: "unkeyed/unkey", defaultBranch: "" } };
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

    const byId = new Map((await loadProjectApps(projectId, undefined)).map((app) => [app.id, app]));

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
