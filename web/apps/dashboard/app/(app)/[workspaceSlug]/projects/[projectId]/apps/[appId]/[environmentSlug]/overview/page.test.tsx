import { cleanup, render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { DeploymentRow } from "../deployments/components/deployment-row";
import Overview from "./page";

vi.mock("@unkey/ui", () => ({
  PageBody: "div",
  PageContainer: "div",
  PageHeader: "header",
  PageHeaderContent: "div",
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
    app: { sourceType: state.sourceType },
    currentDeployment: undefined,
    isRolledBack: false,
  }),
}));
vi.mock("./components/active-branches", () => ({
  ActiveBranches: () => <h2>Active Branches</h2>,
}));
vi.mock("./components/app-production-card", () => ({
  AppProductionCard: () => <div>Production</div>,
}));
vi.mock("./components/overview-page-title", () => ({
  OverviewPageTitle: () => "Container app",
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
