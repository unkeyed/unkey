import { describe, expect, it } from "vitest";
import { buildProjectApps } from "./project-cards";

const app = {
  id: "app_a",
  projectId: "proj_a",
  name: "api",
  updatedAt: 1,
  currentDeploymentId: "dep_current",
};

function deployment(id: string, environmentId: string, createdAt: number) {
  return {
    id,
    appId: app.id,
    environmentId,
    status: "ready" as const,
    gitCommitMessage: null,
    gitBranch: "KEBAP",
    createdAt,
  };
}

function headlineId(
  currentDeploymentId: string | null,
  deployments: ReturnType<typeof deployment>[],
) {
  const [card] = buildProjectApps(app.projectId, {
    apps: [{ ...app, currentDeploymentId }],
    deployments,
    productionDomains: [],
  });
  return card?.headlineDeployment?.id;
}

describe("buildProjectApps headline deployment", () => {
  it("prefers the newest deployment in the current deployment's environment", () => {
    expect(
      headlineId("dep_current", [
        deployment("dep_current", "env_production", 10),
        deployment("dep_newer_production", "env_production", 20),
        deployment("dep_newest_preview", "env_preview", 30),
      ]),
    ).toBe("dep_newer_production");
  });

  it("falls back to the newest deployment in any environment without a current deployment", () => {
    expect(
      headlineId(null, [
        deployment("dep_old", "env_production", 10),
        deployment("dep_newest_preview", "env_preview", 30),
      ]),
    ).toBe("dep_newest_preview");
  });

  it("breaks a createdAt tie with the higher id", () => {
    expect(
      headlineId(null, [
        deployment("dep_a", "env_production", 10),
        deployment("dep_b", "env_production", 10),
      ]),
    ).toBe("dep_b");
  });

  it("has no headline without deployments", () => {
    expect(headlineId(null, [])).toBeUndefined();
  });
});

describe("buildProjectApps app order", () => {
  it("orders by updatedAt descending, never-updated last, ties by higher id", () => {
    const cards = buildProjectApps(app.projectId, {
      apps: [
        { ...app, id: "app_never", updatedAt: null },
        { ...app, id: "app_old", updatedAt: 10 },
        { ...app, id: "app_tie_a", updatedAt: 20 },
        { ...app, id: "app_tie_b", updatedAt: 20 },
        { ...app, id: "app_other_project", projectId: "proj_b", updatedAt: 99 },
      ],
      deployments: [],
      productionDomains: [],
    });
    expect(cards.map((card) => card.id)).toEqual([
      "app_tie_b",
      "app_tie_a",
      "app_old",
      "app_never",
    ]);
  });
});
