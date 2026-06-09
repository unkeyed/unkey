import { describe, expect, it } from "vitest";
import {
  appDeploymentsPath,
  appPath,
  appSettingsPath,
  deploymentPath,
  newAppPath,
  openapiDiffPath,
  projectLogsPath,
  projectPath,
  projectRequestsPath,
  projectSettingsPath,
  projectsPath,
} from "./routes";

const ws = "acme";
const projectId = "proj_123";
const appId = "app_456";
const deploymentId = "dep_789";

describe("project-scoped paths", () => {
  it("builds the list and project base paths", () => {
    expect(projectsPath({ workspaceSlug: ws })).toBe("/acme/projects");
    expect(projectPath({ workspaceSlug: ws, projectId })).toBe("/acme/projects/proj_123");
  });

  it("appends the new-project query when flagged", () => {
    expect(projectsPath({ workspaceSlug: ws, new: true })).toBe("/acme/projects?new=true");
  });

  it("builds project leaf paths", () => {
    const scope = { workspaceSlug: ws, projectId };
    expect(projectSettingsPath(scope)).toBe("/acme/projects/proj_123/settings");
  });
});

describe("projectLogsPath", () => {
  it("omits the query when no app is scoped", () => {
    expect(projectLogsPath({ workspaceSlug: ws, projectId })).toBe("/acme/projects/proj_123/logs");
  });

  it("scopes logs to an app", () => {
    expect(projectLogsPath({ workspaceSlug: ws, projectId, appId })).toBe(
      "/acme/projects/proj_123/logs?appId=app_456",
    );
  });
});

describe("projectRequestsPath", () => {
  it("omits the query when nothing is scoped", () => {
    expect(projectRequestsPath({ workspaceSlug: ws, projectId })).toBe(
      "/acme/projects/proj_123/requests",
    );
  });

  it("builds the since + appId query in order", () => {
    expect(projectRequestsPath({ workspaceSlug: ws, projectId, since: "6h", appId })).toBe(
      "/acme/projects/proj_123/requests?since=6h&appId=app_456",
    );
  });

  it("prefixes a deployment id filter with contains:", () => {
    expect(projectRequestsPath({ workspaceSlug: ws, projectId, since: "6h", deploymentId })).toBe(
      "/acme/projects/proj_123/requests?since=6h&deploymentId=contains:dep_789",
    );
  });
});

describe("newAppPath", () => {
  it("builds the bare new-app path", () => {
    expect(newAppPath({ workspaceSlug: ws, projectId })).toBe("/acme/projects/proj_123/apps/new");
  });

  it("carries the repo-select step and app id", () => {
    expect(newAppPath({ workspaceSlug: ws, projectId, step: "select-repo", appId })).toBe(
      "/acme/projects/proj_123/apps/new?step=select-repo&appId=app_456",
    );
  });
});

describe("app-scoped paths", () => {
  const scope = { workspaceSlug: ws, projectId, appId };

  it("builds app base and leaf paths", () => {
    expect(appPath(scope)).toBe("/acme/projects/proj_123/apps/app_456");
    expect(appSettingsPath(scope)).toBe("/acme/projects/proj_123/apps/app_456/settings");
    expect(appDeploymentsPath(scope)).toBe("/acme/projects/proj_123/apps/app_456/deployments");
  });

  it("builds a deployment path", () => {
    expect(deploymentPath({ ...scope, deploymentId })).toBe(
      "/acme/projects/proj_123/apps/app_456/deployments/dep_789",
    );
  });

  it("flags a build deployment", () => {
    expect(deploymentPath({ ...scope, deploymentId, build: true })).toBe(
      "/acme/projects/proj_123/apps/app_456/deployments/dep_789?build=true",
    );
  });

  it("builds an openapi diff path", () => {
    expect(openapiDiffPath({ ...scope, from: "dep_old", to: "dep_new" })).toBe(
      "/acme/projects/proj_123/apps/app_456/openapi-diff?from=dep_old&to=dep_new",
    );
  });
});
