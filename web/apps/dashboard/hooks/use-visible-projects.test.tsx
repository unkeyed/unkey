import type { Project } from "@/lib/collections/deploy/projects";
import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let projectsNav = false;

const defaultProject: Project = {
  id: "proj_default",
  name: "Default",
  slug: "default",
  isDefault: true,
  apps: [],
  repositoryFullName: null,
  currentDeploymentId: null,
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

const rows = [defaultProject, platformProject];

type Stage = {
  rows: Project[];
  orderBy: (selector: (row: { project: Project }) => number, direction: "asc" | "desc") => Stage;
  where: (predicate: (row: { project: Project }) => boolean) => Stage;
};

function stage(current: Project[]): Stage {
  return {
    rows: current,
    orderBy: (selector, direction) =>
      stage(
        [...current].sort((a, b) => {
          const delta = selector({ project: a }) - selector({ project: b });
          return direction === "desc" ? -delta : delta;
        }),
      ),
    where: (predicate) => stage(current.filter((row) => predicate({ project: row }))),
  };
}

vi.mock("@/lib/flags/provider", () => ({ useFlag: () => projectsNav }));
vi.mock("@/lib/collections", () => ({ collection: { projects: {} } }));
vi.mock("@tanstack/react-db", () => ({
  eq: (left: unknown, right: unknown) => left === right,
  useLiveQuery: (build: (q: { from: () => Stage }) => Stage) => ({
    data: build({ from: () => stage(rows) }).rows,
    isLoading: false,
  }),
}));

import { useVisibleProjects } from "./use-visible-projects";

const visibleIds = () =>
  renderHook(() => useVisibleProjects()).result.current.data.map((project) => project.id);

beforeEach(() => {
  projectsNav = false;
});

describe("useVisibleProjects", () => {
  it("hides the default project with the flag off", () => {
    expect(visibleIds()).toEqual(["proj_platform"]);
  });

  it("includes the default project with the flag on, newest first", () => {
    projectsNav = true;
    expect(visibleIds()).toEqual(["proj_platform", "proj_default"]);
  });
});
