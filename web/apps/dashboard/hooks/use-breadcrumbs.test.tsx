import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectOwner, ProjectResource } from "./use-resource-project-id";

let params: Record<string, string> = {};
let projectsNav = false;
let owner: ProjectOwner = { state: "loading" };
const resourceCalls: (ProjectResource | null)[] = [];

vi.mock("next/navigation", () => ({
  useParams: () => params,
  useSelectedLayoutSegments: () => [],
}));
vi.mock("@/lib/flags/provider", () => ({ useFlag: () => projectsNav }));
vi.mock("./use-workspace-navigation", () => ({
  useWorkspaceNavigation: () => ({ slug: "acme", name: "Acme" }),
}));
vi.mock("./use-resource-project-id", () => ({
  useResourceProjectId: (resource: ProjectResource | null) => {
    resourceCalls.push(resource);
    return resource ? owner : null;
  },
}));

import { useBreadcrumbs } from "./use-breadcrumbs";

const crumbs = () => renderHook(() => useBreadcrumbs()).result.current;

beforeEach(() => {
  params = {};
  projectsNav = false;
  owner = { state: "loading" };
  resourceCalls.length = 0;
});

describe("with the projects-first flag off", () => {
  it("keeps the keyspace crumbs at workspace scope", () => {
    params = { apiId: "api_1" };
    expect(crumbs()).toEqual([
      { type: "workspace", href: "/acme/apis" },
      { type: "api", apiId: "api_1" },
    ]);
  });

  it("looks up no owning project", () => {
    params = { namespaceId: "ns_1" };
    crumbs();
    expect(resourceCalls).toEqual([null]);
  });
});

describe("with the projects-first flag on", () => {
  beforeEach(() => {
    projectsNav = true;
    owner = { state: "resolved", projectId: "proj_1" };
  });

  it.each([
    ["apiId", "api_1", { type: "api", apiId: "api_1" }],
    ["namespaceId", "ns_1", { type: "namespace", namespaceId: "ns_1" }],
    ["identityId", "id_1", { type: "identity", identityId: "id_1" }],
  ])("inserts the owning project for %s", (key, value, leaf) => {
    params = { [key]: value };
    expect(crumbs()).toEqual([
      { type: "workspace", href: "/acme/projects" },
      { type: "project", owner: { state: "resolved", projectId: "proj_1" } },
      leaf,
    ]);
  });

  it.each([
    ["apiId", "api_1", { type: "api", apiId: "api_1" }],
    ["namespaceId", "ns_1", { type: "namespace", namespaceId: "ns_1" }],
    ["identityId", "id_1", { type: "identity", identityId: "id_1" }],
  ])("renders a loading project crumb for %s until the lookup resolves", (key, value, leaf) => {
    params = { [key]: value };
    owner = { state: "loading" };
    expect(crumbs()).toEqual([
      { type: "workspace", href: "/acme/projects" },
      { type: "project", owner: { state: "loading" } },
      leaf,
    ]);
  });

  it("keeps the project crumb slot when the owner cannot be resolved", () => {
    params = { identityId: "id_gone" };
    owner = { state: "unknown" };
    expect(crumbs()).toEqual([
      { type: "workspace", href: "/acme/projects" },
      { type: "project", owner: { state: "unknown" } },
      { type: "identity", identityId: "id_gone" },
    ]);
  });

  it("reads Workspace then Project on a project-mounted page, with no lookup", () => {
    params = { projectId: "proj_2" };
    expect(crumbs()).toEqual([
      { type: "workspace", href: "/acme/projects" },
      { type: "project", owner: { state: "resolved", projectId: "proj_2" } },
    ]);
    expect(resourceCalls).toEqual([null]);
  });

  it.each([
    ["apiId", "api_1", "api"],
    ["namespaceId", "ns_1", "namespace"],
    ["identityId", "id_1", "identity"],
  ])("scopes the %s crumb to the project in the url, with no lookup", (key, value, type) => {
    params = { projectId: "proj_2", [key]: value };
    expect(crumbs()).toEqual([
      { type: "workspace", href: "/acme/projects" },
      { type: "project", owner: { state: "resolved", projectId: "proj_2" } },
      { type, [key]: value, projectId: "proj_2" },
    ]);
    expect(resourceCalls).toEqual([null]);
  });
});
