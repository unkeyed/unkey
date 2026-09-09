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

const state = vi.hoisted(() => ({
  sourceType: "oci",
  deployments: Array.from({ length: 6 }, (_, index) => ({
    id: `d_${6 - index}`,
    environmentId: "env_preview",
    status: "stopped",
  })),
}));

vi.mock("@/hooks/use-workspace-navigation", () => ({
  useWorkspaceNavigation: () => ({ slug: "workspace" }),
}));
vi.mock("../data-provider", () => ({
  useAppId: () => "app_container",
  useProjectData: () => ({
    projectId: "proj_backend",
    deployments: state.deployments,
    environments: [{ id: "env_preview", slug: "preview" }],
    isDeploymentsLoading: false,
  }),
}));
vi.mock("../hooks/use-app-current-deployment", () => ({
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
  state.sourceType = "oci";
});

describe("Overview deployment history", () => {
  it("shows the five most recent container deployments, including stopped deployments", () => {
    render(<Overview />);
    expect(screen.getByRole("heading", { name: "Recent Deployments" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Active Branches" })).toBeNull();
    expect(screen.getAllByRole("listitem").map((row) => row.textContent)).toEqual([
      "d_6",
      "d_5",
      "d_4",
      "d_3",
      "d_2",
    ]);
    expect(screen.getByRole("link", { name: "d_6" }).getAttribute("href")).toBe(
      "/workspace/projects/proj_backend/apps/app_container/deployments/d_6",
    );
    expect(screen.getByRole("link", { name: "View all deployments" }).getAttribute("href")).toBe(
      "/workspace/projects/proj_backend/apps/app_container/deployments",
    );
  });

  it.each(["git", "unknown"])("preserves the existing branch view for %s apps", (source) => {
    state.sourceType = source;
    render(<Overview />);
    expect(screen.getByRole("heading", { name: "Active Branches" })).toBeTruthy();
    expect(screen.queryByRole("heading", { name: "Recent Deployments" })).toBeNull();
  });
});
