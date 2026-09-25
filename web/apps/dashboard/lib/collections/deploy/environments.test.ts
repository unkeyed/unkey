import { and, createLiveQueryCollection, eq, inArray } from "@tanstack/react-db";
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

const { environments } = await import("./environments");

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
});
