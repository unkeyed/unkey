import type { App } from "@/lib/collections/deploy/apps";
import { describe, expect, test } from "vitest";
import { appSource, filterApps, toAppRow } from "./app-row-model";

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
    domain: null,
    headlineDeployment: null,
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

describe("toAppRow", () => {
  const deployment = {
    id: "d_1",
    status: "ready",
    deployedAt: 0,
    commitMessage: "fix: thing",
    commitSha: "abc123",
    branch: "main",
    prNumber: null,
    forkRepositoryFullName: null,
  } satisfies NonNullable<App["headlineDeployment"]>;
  const row = (overrides: Partial<App>) => toAppRow(app(overrides), "/");

  test("links a git deployment to its commit and branch", () => {
    const { commitUrl, branchUrl } = row({
      repositoryFullName: "unkey/api",
      headlineDeployment: deployment,
    });
    expect(commitUrl).toBe("https://github.com/unkey/api/commit/abc123");
    expect(branchUrl).toBe("https://github.com/unkey/api/tree/main");
  });

  test("links a fork PR to the base repo PR and the fork branch", () => {
    const { commitUrl, branchUrl } = row({
      repositoryFullName: "unkey/api",
      headlineDeployment: { ...deployment, prNumber: 42, forkRepositoryFullName: "dev/api" },
    });
    expect(commitUrl).toBe("https://github.com/unkey/api/pull/42");
    expect(branchUrl).toBe("https://github.com/dev/api/tree/main");
  });

  test("has no links for a git app without a repo connection", () => {
    const { source, commitUrl, branchUrl } = row({ headlineDeployment: deployment });
    expect(source).toBe("git");
    expect(commitUrl).toBeUndefined();
    expect(branchUrl).toBeUndefined();
  });

  test("has no links for image and legacy apps or apps never deployed", () => {
    for (const overrides of [
      { sourceType: "oci", headlineDeployment: deployment },
      { sourceType: "unknown", headlineDeployment: deployment },
      { repositoryFullName: "unkey/api" },
    ] satisfies Partial<App>[]) {
      const { commitUrl, branchUrl } = row(overrides);
      expect(commitUrl).toBeUndefined();
      expect(branchUrl).toBeUndefined();
    }
  });
});
