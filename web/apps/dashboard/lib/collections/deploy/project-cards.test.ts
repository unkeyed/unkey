import { describe, expect, it } from "vitest";
import { byLatestUpdate, pickPrimaryApp } from "./project-cards";

const app = { id: "app_a", updatedAt: 1, currentDeploymentId: null };

describe("byLatestUpdate", () => {
  it("orders by updatedAt descending, never-updated last, ties by higher id", () => {
    const apps = [
      { ...app, id: "app_never", updatedAt: null },
      { ...app, id: "app_old", updatedAt: 10 },
      { ...app, id: "app_tie_a", updatedAt: 20 },
      { ...app, id: "app_tie_b", updatedAt: 20 },
    ];
    expect(apps.toSorted(byLatestUpdate).map((a) => a.id)).toEqual([
      "app_tie_b",
      "app_tie_a",
      "app_old",
      "app_never",
    ]);
  });
});

describe("pickPrimaryApp", () => {
  it("picks the most recently updated app with a current deployment", () => {
    const primary = pickPrimaryApp([
      { ...app, id: "app_newest_undeployed", updatedAt: 30 },
      { ...app, id: "app_deployed_old", updatedAt: 10, currentDeploymentId: "dep_old" },
      { ...app, id: "app_deployed_new", updatedAt: 20, currentDeploymentId: "dep_KEBAP" },
    ]);
    expect(primary?.id).toBe("app_deployed_new");
  });

  it("has no primary app when nothing is deployed", () => {
    expect(pickPrimaryApp([app])).toBeUndefined();
  });
});
