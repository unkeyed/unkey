import type { App } from "@/lib/collections/deploy/apps";
import { describe, expect, test } from "vitest";
import { appSource, filterApps } from "./app-row-model";

function app(overrides: Partial<App>): App {
  return {
    id: "app_1",
    projectId: "proj_1",
    name: "api",
    slug: "api",
    sourceType: "git",
    imageReference: null,
    defaultBranch: "main",
    currentDeploymentId: null,
    isRolledBack: false,
    updatedAt: null,
    repositoryFullName: null,
    latestDeploymentId: null,
    commitTitle: null,
    commitSha: null,
    forkRepositoryFullName: null,
    prNumber: null,
    branch: "main",
    author: null,
    authorAvatar: null,
    commitTimestamp: null,
    domain: null,
    ...overrides,
  };
}

describe("appSource", () => {
  test("maps source types to how the app is shown", () => {
    expect(appSource(app({ sourceType: "git" }))).toBe("git");
    expect(appSource(app({ sourceType: "oci" }))).toBe("image");
    expect(appSource(app({ sourceType: "unknown", repositoryFullName: "unkey/api" }))).toBe("git");
    expect(appSource(app({ sourceType: "unknown" }))).toBe("legacy");
  });
});

describe("filterApps", () => {
  const rows = [
    { app: app({ id: "a", name: "Gateway", domain: "gw.unkey.app" }) },
    { app: app({ id: "b", name: "worker", repositoryFullName: "unkey/Jobs" }) },
    { app: app({ id: "c", name: "vault", imageReference: "hashicorp/vault:1.17" }) },
  ];
  const ids = (search: string) => filterApps(rows, search).map((row) => row.app.id);

  test("returns every row for a blank query", () => {
    expect(ids("")).toEqual(["a", "b", "c"]);
    expect(ids("   ")).toEqual(["a", "b", "c"]);
  });

  test("matches name, domain, repository and image case-insensitively", () => {
    expect(ids("gateway")).toEqual(["a"]);
    expect(ids("GW.UNKEY")).toEqual(["a"]);
    expect(ids("jobs")).toEqual(["b"]);
    expect(ids("hashicorp")).toEqual(["c"]);
  });

  test("returns nothing when no field matches", () => {
    expect(ids("nope")).toEqual([]);
  });
});
