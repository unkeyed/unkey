import { cleanup, render, screen } from "@testing-library/react";
import type { ComponentProps, ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { DeploymentRow } from "../deployments/components/deployment-row";
import Overview from "./page";

vi.mock("@unkey/ui", () => ({
  Button: ({ children, onClick }: { children: ReactNode; onClick?: () => void }) => (
    <button type="button" onClick={onClick}>
      {children}
    </button>
  ),
  PageBody: "div",
  PageContainer: "div",
  PageHeader: "header",
  PageHeaderActions: "div",
  PageHeaderContent: "div",
  PageHeaderDescription: "p",
  PageHeaderTitle: "h1",
  ResourceList: "section",
  ResourceListBody: "ul",
  ResourceListContent: "div",
  ResourceListHeader: "header",
}));

const sixDeployments = (environmentId: string) =>
  Array.from({ length: 6 }, (_, index) => ({
    id: `d_${6 - index}`,
    environmentId,
    status: "stopped",
  }));

const state = vi.hoisted(() => ({
  sourceType: "git",
  kind: "production",
  deployments: [] as { id: string; environmentId: string; status: string }[],
  overview: { kind: "loading" } as Record<string, unknown>,
}));

vi.mock("@/hooks/use-workspace-navigation", () => ({
  useWorkspaceNavigation: () => ({ slug: "workspace" }),
}));
vi.mock("../../data-provider", () => ({
  useAppId: () => "app_container",
  useProjectData: () => ({
    projectId: "proj_backend",
    deployments: state.deployments,
    environments: [{ id: `env_${state.kind}`, slug: state.kind }],
    isDeploymentsLoading: false,
  }),
}));
vi.mock("../environment-context", () => ({
  useAppScope: () => ({
    workspaceSlug: "workspace",
    projectId: "proj_backend",
    appId: "app_container",
    environmentSlug: state.kind,
  }),
  useAppEnvironment: () => ({
    environment: { id: `env_${state.kind}`, slug: state.kind, kind: state.kind },
  }),
}));
vi.mock("../../hooks/use-app-current-deployment", () => ({
  useAppCurrentDeployment: () => ({
    app: {
      name: "Container app",
      sourceType: state.sourceType,
      repositoryFullName: "acme/api",
      defaultBranch: "main",
      imageReference: "ghcr.io/acme/api:latest",
    },
    currentDeployment: undefined,
    isRolledBack: false,
  }),
}));
vi.mock("./components/active-branches", () => ({
  ActiveBranches: () => <h2>Active Branches</h2>,
}));
vi.mock("./components/use-app-overview", () => ({
  useAppOverview: () => state.overview,
}));
vi.mock("./components/app-production-card", () => ({
  AppProductionCard: () => <div>Production</div>,
}));
vi.mock("./components/app-production-card-skeleton", () => ({
  AppProductionCardSkeleton: () => <div>Loading</div>,
}));
vi.mock("./components/environment-pending-card", () => ({
  EnvironmentPendingCard: () => <div>Pending</div>,
}));
vi.mock("./components/production-card-actions-menu", () => ({
  ProductionCardActionsMenu: () => <button type="button">More actions</button>,
}));
vi.mock("../navigations/create-deployment-button", () => ({
  CreateDeploymentButton: () => <button type="button">Create deployment</button>,
}));
vi.mock("../deployments/components/deployment-row", () => ({
  DeploymentRow: ({ deployment, href }: ComponentProps<typeof DeploymentRow>) => (
    <li>
      <a href={href}>{deployment.id}</a>
    </li>
  ),
}));

afterEach(cleanup);
beforeEach(() => {
  state.sourceType = "git";
  state.kind = "production";
  state.deployments = sixDeployments("env_production");
  state.overview = { kind: "loading" };
});

const deployedOverview = ({ rollbackTarget }: { rollbackTarget?: { id: string } }) => ({
  kind: "deployed",
  domains: [
    { hostname: "api.acme.com", url: "https://api.acme.com", source: "custom" },
    { hostname: "api-acme.unkey.app", url: "https://api-acme.unkey.app", source: "platform" },
  ],
  newerDeployment: undefined,
  dialogs: null,
  card: {
    deployment: {
      id: "d_6",
      source: "git",
      gitBranch: "feat/apple-pay",
      gitCommitSha: "abc1234",
      forkRepositoryFullName: null,
    },
    sourceRepo: "acme/api",
    status: "ready",
    diagnostic: null,
    isRolledBack: false,
    rollbackTarget,
    openRollback: () => undefined,
    deploymentHref: "/d_6",
    logsHref: "/logs",
    requestsHref: "/requests",
  },
});

describe("Overview header", () => {
  it("shows the app, the environment and the default branch before the first deployment", () => {
    state.overview = { kind: "pending", newerDeployment: undefined };
    render(<Overview />);
    expect(screen.getByRole("heading", { level: 1 }).textContent).toBe("Container appproduction");
    expect(screen.getByRole("link", { name: "acme/api:main" }).getAttribute("href")).toBe(
      "https://github.com/acme/api/tree/main",
    );
    expect(screen.getByRole("button", { name: "Create deployment" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Instant Rollback" })).toBeNull();
  });

  it("shows the deployed branch, the primary domain and rollback when eligible", () => {
    state.overview = deployedOverview({ rollbackTarget: { id: "d_5" } });
    render(<Overview />);
    expect(screen.getByRole("link", { name: "acme/api:feat/apple-pay" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "api.acme.com" }).getAttribute("href")).toBe(
      "https://api.acme.com",
    );
    expect(screen.queryByText("api-acme.unkey.app")).toBeNull();
    expect(screen.getByRole("button", { name: "Instant Rollback" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "More actions" })).toBeTruthy();
  });

  it("hides rollback without an eligible target", () => {
    state.overview = deployedOverview({});
    render(<Overview />);
    expect(screen.queryByRole("button", { name: "Instant Rollback" })).toBeNull();
    expect(screen.getByRole("button", { name: "More actions" })).toBeTruthy();
  });

  it("shows the image for container apps", () => {
    state.sourceType = "oci";
    state.overview = { kind: "pending", newerDeployment: undefined };
    render(<Overview />);
    expect(screen.getByText("ghcr.io/acme/api:latest")).toBeTruthy();
  });
});

describe("Overview history section", () => {
  it("shows the five latest production deployments with a link to all of them", () => {
    render(<Overview />);
    expect(screen.getByRole("heading", { name: "Latest deployments" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Active Branches" })).toBeNull();
    expect(screen.getAllByRole("listitem").map((row) => row.textContent)).toEqual([
      "d_6",
      "d_5",
      "d_4",
      "d_3",
      "d_2",
    ]);
    expect(screen.getByRole("link", { name: "d_6" }).getAttribute("href")).toBe(
      "/workspace/projects/proj_backend/apps/app_container/production/deployments/d_6",
    );
    expect(screen.getByRole("link", { name: "View all deployments" }).getAttribute("href")).toBe(
      "/workspace/projects/proj_backend/apps/app_container/production/deployments",
    );
  });

  it("shows an empty state when production has no deployments", () => {
    state.deployments = sixDeployments("env_preview");
    render(<Overview />);
    expect(screen.getByRole("heading", { name: "Latest deployments" })).toBeTruthy();
    expect(screen.getByText("No deployments yet.")).toBeTruthy();
  });

  it("shows active branches in preview", () => {
    state.kind = "preview";
    state.deployments = sixDeployments("env_preview");
    render(<Overview />);
    expect(screen.getByRole("heading", { name: "Active Branches" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Latest deployments" })).toBeNull();
  });

  it("shows latest deployments in preview for container apps, which have no branches", () => {
    state.sourceType = "oci";
    state.kind = "preview";
    state.deployments = sixDeployments("env_preview");
    render(<Overview />);
    expect(screen.getByRole("heading", { name: "Latest deployments" })).toBeTruthy();
    expect(screen.getAllByRole("listitem")).toHaveLength(5);
  });
});
