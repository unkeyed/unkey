import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";
const projectId = "proj_123";
const appId = "app_456";
const deploymentId = "d_789";

describe("project-scoped paths", () => {
  it("builds the list and project base paths", () => {
    expect(routes.projects.list({ workspaceSlug: ws })).toBe("/acme/projects");
    expect(routes.projects.detail({ workspaceSlug: ws, projectId })).toBe(
      "/acme/projects/proj_123",
    );
  });

  it("appends the new-project query when flagged", () => {
    expect(routes.projects.list({ workspaceSlug: ws, new: true })).toBe("/acme/projects?new=true");
  });

  it("builds project leaf paths", () => {
    const scope = { workspaceSlug: ws, projectId };
    expect(routes.projects.settings(scope)).toBe("/acme/projects/proj_123/settings");
  });
});

describe("routes.projects.logs", () => {
  it("omits the query when no app is scoped", () => {
    expect(routes.projects.logs({ workspaceSlug: ws, projectId })).toBe(
      "/acme/projects/proj_123/logs",
    );
  });

  it("scopes logs to an app, prefixing the app id filter with is:", () => {
    expect(routes.projects.logs({ workspaceSlug: ws, projectId, appId })).toBe(
      "/acme/projects/proj_123/logs?appId=is:app_456",
    );
  });

  it("scopes logs to one instance of a deployment", () => {
    expect(
      routes.projects.logs({ workspaceSlug: ws, projectId, deploymentId, instanceId: "ins_1" }),
    ).toBe("/acme/projects/proj_123/logs?deploymentId=is:d_789&instanceId=is:ins_1");
  });

  it("joins several deployment and instance ids into one any-of filter each", () => {
    expect(
      routes.projects.logs({
        workspaceSlug: ws,
        projectId,
        appId,
        deploymentId: ["d_1", "d_2"],
        instanceId: ["ins_1", "ins_2"],
      }),
    ).toBe(
      "/acme/projects/proj_123/logs?appId=is:app_456&deploymentId=is:d_1,is:d_2&instanceId=is:ins_1,is:ins_2",
    );
  });

  it("drops a filter given an empty list", () => {
    expect(
      routes.projects.logs({ workspaceSlug: ws, projectId, deploymentId: [], instanceId: [] }),
    ).toBe("/acme/projects/proj_123/logs");
  });

  it("scopes logs to a time window", () => {
    expect(
      routes.projects.logs({
        workspaceSlug: ws,
        projectId,
        deploymentId,
        startTime: 1_000,
        endTime: 2_000,
      }),
    ).toBe("/acme/projects/proj_123/logs?deploymentId=is:d_789&startTime=1000&endTime=2000");
  });
});

describe("routes.projects.requests", () => {
  it("omits the query when nothing is scoped", () => {
    expect(routes.projects.requests({ workspaceSlug: ws, projectId })).toBe(
      "/acme/projects/proj_123/requests",
    );
  });

  it("builds the since + appId query in order, prefixing the app id filter with is:", () => {
    expect(routes.projects.requests({ workspaceSlug: ws, projectId, since: "6h", appId })).toBe(
      "/acme/projects/proj_123/requests?since=6h&appId=is:app_456",
    );
  });

  it("prefixes a deployment id filter with is:", () => {
    expect(
      routes.projects.requests({ workspaceSlug: ws, projectId, since: "6h", deploymentId }),
    ).toBe("/acme/projects/proj_123/requests?since=6h&deploymentId=is:d_789");
  });
});

describe("routes.projects.apps.new", () => {
  it("builds the bare new-app path", () => {
    expect(routes.projects.apps.new({ workspaceSlug: ws, projectId })).toBe(
      "/acme/projects/proj_123/apps/new",
    );
  });

  it("carries the repo-select step and app id", () => {
    expect(
      routes.projects.apps.new({ workspaceSlug: ws, projectId, step: "select-repo", appId }),
    ).toBe("/acme/projects/proj_123/apps/new?step=select-repo&appId=app_456");
  });
});

describe("app-scoped paths", () => {
  const scope = { workspaceSlug: ws, projectId, appId };

  it("builds app leaf paths", () => {
    expect(routes.projects.apps.settings(scope)).toBe(
      "/acme/projects/proj_123/apps/app_456/settings",
    );
    expect(routes.projects.apps.deployments(scope)).toBe(
      "/acme/projects/proj_123/apps/app_456/deployments",
    );
  });

  it("builds a deployment path", () => {
    expect(routes.projects.apps.deployment({ ...scope, deploymentId })).toBe(
      "/acme/projects/proj_123/apps/app_456/deployments/d_789",
    );
  });

  it("flags a build deployment", () => {
    expect(routes.projects.apps.deployment({ ...scope, deploymentId, build: true })).toBe(
      "/acme/projects/proj_123/apps/app_456/deployments/d_789?build=true",
    );
  });

  it("asks the domains and policies pages to open their add flow", () => {
    expect(routes.projects.apps.domains({ ...scope, add: true })).toBe(
      "/acme/projects/proj_123/apps/app_456/domains?add=true",
    );
    expect(routes.projects.apps.policies({ ...scope, add: true })).toBe(
      "/acme/projects/proj_123/apps/app_456/policies?add=true",
    );
    expect(routes.projects.apps.policies({ ...scope, add: false })).toBe(
      "/acme/projects/proj_123/apps/app_456/policies",
    );
  });

  it("carries an env vars search", () => {
    expect(routes.projects.apps.envVars(scope)).toBe(
      "/acme/projects/proj_123/apps/app_456/env-vars",
    );
    expect(routes.projects.apps.envVars({ ...scope, search: "REDIS_URL" })).toBe(
      "/acme/projects/proj_123/apps/app_456/env-vars?search=REDIS_URL",
    );
  });

  it("builds an openapi diff path", () => {
    expect(routes.projects.apps.openapiDiff({ ...scope, from: "dep_old", to: "dep_new" })).toBe(
      "/acme/projects/proj_123/apps/app_456/openapi-diff?from=dep_old&to=dep_new",
    );
  });
});
