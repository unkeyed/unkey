import { newId } from "@unkey/id";
import { describe, expect, it, vi } from "vitest";

const listProjects = vi.fn();
const getDefault = vi.fn();
vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ projects: { listProjects } }),
  getErrorToast: vi.fn(),
}));
vi.mock("@/lib/collections/client", () => ({
  trpcClient: { deploy: { project: { getDefault: { query: getDefault } } } },
}));

const { loadProjects, visibleProjects } = await import("./projects");
type Project = Awaited<ReturnType<typeof loadProjects>>[number];

function project(overrides: Partial<Project>): Project {
  return {
    id: newId("project"),
    name: "KEBAP",
    slug: "kebap",
    isDefault: false,
    createdAt: 1,
    ...overrides,
  };
}

describe("visibleProjects", () => {
  const defaultProject = project({ isDefault: true, createdAt: 1 });
  const older = project({ createdAt: 2 });
  const newer = project({ createdAt: 3 });
  const all = [defaultProject, older, newer];

  it("hides the default project unless asked to include it", () => {
    expect(visibleProjects(all, false).map((p) => p.id)).toEqual([newer.id, older.id]);
  });

  it("lists every project newest first when the default one is included", () => {
    expect(visibleProjects(all, true).map((p) => p.id)).toEqual([
      newer.id,
      older.id,
      defaultProject.id,
    ]);
  });
});

describe("loadProjects", () => {
  it("puts the default project first, then every page of the SDK list", async () => {
    const pages = [
      [{ id: newId("project"), name: "api", slug: "api", createdAt: 1 }],
      [{ id: newId("project"), name: "KEBAP", slug: "kebap", createdAt: 2 }],
    ];
    listProjects.mockResolvedValue({
      async *[Symbol.asyncIterator]() {
        for (const data of pages) {
          yield { result: { data } };
        }
      },
    });
    const defaultProject = { id: newId("project"), name: "Default", slug: "default", createdAt: 0 };

    getDefault.mockResolvedValue(defaultProject);

    expect(await loadProjects()).toEqual([
      { ...defaultProject, isDefault: true },
      ...pages.flat().map((p) => ({ ...p, isDefault: false })),
    ]);
    expect(listProjects).toHaveBeenCalledWith({ limit: 100 }, { signal: undefined });
  });
});
