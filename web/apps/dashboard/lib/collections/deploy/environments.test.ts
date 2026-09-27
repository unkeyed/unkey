import { IR, and, createLiveQueryCollection, eq, gt, inArray, or } from "@tanstack/react-db";

const { PropRef } = IR;
import { describe, expect, it, vi } from "vitest";

const listEnvironments = vi.fn(async ({ app }: { project: string; app: string }) => ({
  data: [
    { id: `env_prod_${app}`, slug: "production", kind: "production" as const },
    { id: `env_prev_${app}`, slug: "preview", kind: "preview" as const },
  ],
}));

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ environments: { listEnvironments } }),
}));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return { queryClient: new QueryClient() };
});

const { appsInWhere, environments, inProjectApps } = await import("./environments");

function sortedCalls() {
  return listEnvironments.mock.calls.map(([c]) => c).sort((x, y) => x.app.localeCompare(y.app));
}

describe("environments collection", () => {
  it("lists the environments of every app in the filter, each with its project", async () => {
    listEnvironments.mockClear();
    const query = createLiveQueryCollection((q) =>
      q
        .from({ env: environments })
        .where(({ env }) =>
          and(eq(env.projectId, "proj_1"), inArray(env.appId, ["app_a", "app_b"])),
        ),
    );
    await query.preload();

    await vi.waitFor(() => expect(query.toArray).toHaveLength(4));
    expect(query.toArray.map((e) => e.appId).sort()).toEqual(["app_a", "app_a", "app_b", "app_b"]);
    expect(query.toArray.every((e) => e.projectId === "proj_1")).toBe(true);
    expect(sortedCalls()).toEqual([
      { project: "proj_1", app: "app_a" },
      { project: "proj_1", app: "app_b" },
    ]);
  });

  it("lists one app's environments", async () => {
    listEnvironments.mockClear();
    const query = createLiveQueryCollection((q) =>
      q
        .from({ env: environments })
        .where(({ env }) => and(eq(env.projectId, "proj_2"), eq(env.appId, "app_c"))),
    );
    await query.preload();

    await vi.waitFor(() => expect(query.toArray).toHaveLength(2));
    expect(sortedCalls()).toEqual([{ project: "proj_2", app: "app_c" }]);
  });

  it("lists the environments of apps across projects", async () => {
    listEnvironments.mockClear();
    const projects = [
      { id: "proj_4", apps: [{ id: "app_d" }, { id: "app_e" }] },
      { id: "proj_5", apps: [{ id: "app_f" }] },
    ];
    const query = createLiveQueryCollection((q) =>
      q.from({ env: environments }).where(({ env }) => inProjectApps(env, projects)),
    );
    await query.preload();

    await vi.waitFor(() => expect(query.toArray).toHaveLength(6));
    expect(sortedCalls()).toEqual([
      { project: "proj_4", app: "app_d" },
      { project: "proj_4", app: "app_e" },
      { project: "proj_5", app: "app_f" },
    ]);
  });
});

describe("appsInWhere", () => {
  const env = { projectId: new PropRef(["projectId"]), appId: new PropRef(["appId"]) };

  it("pairs every app with its project", () => {
    expect(
      appsInWhere(
        or(
          and(eq(env.projectId, "proj_1"), inArray(env.appId, ["app_a", "app_b"])),
          and(eq(env.projectId, "proj_2"), eq(env.appId, "app_c")),
        ),
      ),
    ).toEqual([
      { projectId: "proj_1", appId: "app_a" },
      { projectId: "proj_1", appId: "app_b" },
      { projectId: "proj_2", appId: "app_c" },
    ]);
  });

  it("ignores predicates on other fields", () => {
    expect(
      appsInWhere(
        and(eq(env.projectId, "proj_1"), eq(env.appId, "KEBAP"), gt(new PropRef(["slug"]), "a")),
      ),
    ).toEqual([{ projectId: "proj_1", appId: "KEBAP" }]);
  });

  it("keeps only the apps both appId filters allow", () => {
    expect(
      appsInWhere(
        and(
          eq(env.projectId, "proj_1"),
          inArray(env.appId, ["app_a", "app_b"]),
          eq(env.appId, "app_b"),
        ),
      ),
    ).toEqual([{ projectId: "proj_1", appId: "app_b" }]);
  });

  it("rejects a branch without a project or without apps", () => {
    expect(appsInWhere(inArray(env.appId, ["app_a"]))).toBeNull();
    expect(appsInWhere(eq(env.projectId, "proj_1"))).toBeNull();
    expect(
      appsInWhere(
        or(and(eq(env.projectId, "proj_1"), eq(env.appId, "app_a")), eq(env.appId, "app_b")),
      ),
    ).toBeNull();
    expect(appsInWhere(undefined)).toBeNull();
  });
});
