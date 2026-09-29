import { describe, expect, it, vi } from "vitest";

const pages = [
  [{ id: "proj_a", name: "api", slug: "api", createdAt: 1 }],
  [{ id: "proj_b", name: "KEBAP", slug: "kebap", createdAt: 2 }],
];

const listProjects = vi.fn(async () => ({
  async *[Symbol.asyncIterator]() {
    for (const data of pages) {
      yield { result: { data } };
    }
  },
}));
const getDefault = vi.fn(async () => ({
  id: "proj_default",
  name: "Default",
  slug: "default",
  createdAt: 0,
}));

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ projects: { listProjects } }),
  getErrorToast: vi.fn(),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: vi.fn() } }));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return {
    queryClient: new QueryClient(),
    trpcClient: { deploy: { project: { getDefault: { query: getDefault } } } },
  };
});

const { projects } = await import("./projects");

describe("projects collection", () => {
  it("lists the default project and every page of the SDK list", async () => {
    await projects.preload();

    expect(projects.toArray.toSorted((a, b) => a.createdAt - b.createdAt)).toEqual([
      { id: "proj_default", name: "Default", slug: "default", isDefault: true, createdAt: 0 },
      { id: "proj_a", name: "api", slug: "api", isDefault: false, createdAt: 1 },
      { id: "proj_b", name: "KEBAP", slug: "kebap", isDefault: false, createdAt: 2 },
    ]);
    expect(listProjects).toHaveBeenCalledWith({ limit: 100 });
  });
});
