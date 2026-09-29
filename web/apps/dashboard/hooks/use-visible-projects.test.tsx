import type { Project } from "@/lib/collections/deploy/projects";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let projectsNav = false;

const { rows } = vi.hoisted(() => {
  const defaultProject: Project = {
    id: "proj_default",
    name: "Default",
    slug: "default",
    isDefault: true,
    createdAt: 1,
  };
  const platformProject: Project = {
    ...defaultProject,
    id: "proj_platform",
    name: "Platform",
    slug: "platform",
    isDefault: false,
    createdAt: 2,
  };
  return { rows: [defaultProject, platformProject] };
});

vi.mock("@/lib/flags/provider", () => ({ useFlag: () => projectsNav }));
// A real in-memory collection, so the query runs through the actual builder
// instead of a hand-written stand-in for `where` and `eq`.
vi.mock("@/lib/collections", async () => {
  const { createCollection, localOnlyCollectionOptions } = await import("@tanstack/react-db");
  return {
    collection: {
      projects: createCollection(
        localOnlyCollectionOptions<Project, string>({
          getKey: (project) => project.id,
          initialData: rows,
        }),
      ),
    },
  };
});

import { useVisibleProjects } from "./use-visible-projects";

async function visibleIds(): Promise<string[]> {
  const { result } = renderHook(() => useVisibleProjects());
  await waitFor(() => expect(result.current.isLoading).toBe(false));
  return result.current.data.map((project) => project.id);
}

beforeEach(() => {
  projectsNav = false;
});

describe("useVisibleProjects", () => {
  it("hides the default project with the flag off", async () => {
    expect(await visibleIds()).toEqual(["proj_platform"]);
  });

  it("includes the default project with the flag on, newest first", async () => {
    projectsNav = true;
    expect(await visibleIds()).toEqual(["proj_platform", "proj_default"]);
  });
});
