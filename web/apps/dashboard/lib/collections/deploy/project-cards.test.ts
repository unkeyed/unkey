import { describe, expect, it } from "vitest";
import { buildProjectApps } from "./project-cards";

const app = {
  id: "app_a",
  projectId: "proj_a",
  name: "api",
  updatedAt: 1,
  currentDeploymentId: null,
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
  current: ReturnType<typeof deployment> | undefined,
  recentDeployments: ReturnType<typeof deployment>[],
) {
  const [card] = buildProjectApps(app.projectId, {
    apps: [{ app: { ...app, currentDeploymentId: current?.id ?? null }, current }],
    recentDeployments,
    productionDomains: [],
  });
  return card?.headlineDeployment?.id;
}

describe("buildProjectApps headline deployment", () => {
  it("prefers the newest deployment in the current deployment's environment", () => {
    const current = deployment("dep_current", "env_production", 10);
    expect(
      headlineId(current, [
        current,
        deployment("dep_newer_production", "env_production", 20),
        deployment("dep_newest_preview", "env_preview", 30),
      ]),
    ).toBe("dep_newer_production");
  });

  it("uses a current deployment outside the recent window", () => {
    expect(
      headlineId(deployment("dep_current", "env_production", 10), [
        deployment("dep_newest_preview", "env_preview", 30),
      ]),
    ).toBe("dep_current");
  });

  it("falls back to the newest deployment in any environment without a current deployment", () => {
    expect(
      headlineId(undefined, [
        deployment("dep_old", "env_production", 10),
        deployment("dep_newest_preview", "env_preview", 30),
      ]),
    ).toBe("dep_newest_preview");
  });

  it("breaks a createdAt tie with the higher id", () => {
    expect(
      headlineId(undefined, [
        deployment("dep_a", "env_production", 10),
        deployment("dep_b", "env_production", 10),
      ]),
    ).toBe("dep_b");
  });

  it("has no headline without deployments", () => {
    expect(headlineId(undefined, [])).toBeUndefined();
  });
});

describe("buildProjectApps app order", () => {
  it("orders by updatedAt descending, never-updated last, ties by higher id", () => {
    const cards = buildProjectApps(app.projectId, {
      apps: [
        { app: { ...app, id: "app_never", updatedAt: null } },
        { app: { ...app, id: "app_old", updatedAt: 10 } },
        { app: { ...app, id: "app_tie_a", updatedAt: 20 } },
        { app: { ...app, id: "app_tie_b", updatedAt: 20 } },
        { app: { ...app, id: "app_other_project", projectId: "proj_b", updatedAt: 99 } },
      ],
      recentDeployments: [],
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
