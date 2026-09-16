import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectResource } from "./use-resource-project-id";

let params: Record<string, string> = {};
let projectsNav = false;
let ownerProjectId: string | null = null;
const resourceCalls: (ProjectResource | null)[] = [];

vi.mock("next/navigation", () => ({ useParams: () => params }));
vi.mock("@/lib/flags/provider", () => ({ useFlag: () => projectsNav }));
vi.mock("./use-workspace-navigation", () => ({
  useWorkspaceNavigation: () => ({ slug: "acme", name: "Acme" }),
}));
vi.mock("./use-resource-project-id", () => ({
  useResourceProjectId: (resource: ProjectResource | null) => {
    resourceCalls.push(resource);
    return resource ? ownerProjectId : null;
  },
}));

import { useBreadcrumbs } from "./use-breadcrumbs";

const crumbs = () => renderHook(() => useBreadcrumbs()).result.current;

beforeEach(() => {
  params = {};
  projectsNav = false;
  ownerProjectId = null;
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
    ownerProjectId = "proj_1";
  });

  it.each([
    ["apiId", "api_1", { type: "api", apiId: "api_1" }],
    ["namespaceId", "ns_1", { type: "namespace", namespaceId: "ns_1" }],
    ["identityId", "id_1", { type: "identity", identityId: "id_1" }],
  ])("inserts the owning project for %s", (key, value, leaf) => {
    params = { [key]: value };
    expect(crumbs()).toEqual([
      { type: "workspace", href: "/acme/projects" },
      { type: "project", projectId: "proj_1" },
      leaf,
    ]);
  });

  it("omits the project crumb until the lookup resolves", () => {
    params = { apiId: "api_1" };
    ownerProjectId = null;
    expect(crumbs()).toEqual([
      { type: "workspace", href: "/acme/apis" },
      { type: "api", apiId: "api_1" },
    ]);
  });

  it("reads Workspace then Project on a project-mounted page, with no lookup", () => {
    params = { projectId: "proj_2" };
    expect(crumbs()).toEqual([
      { type: "workspace", href: "/acme/projects" },
      { type: "project", projectId: "proj_2" },
    ]);
    expect(resourceCalls).toEqual([null]);
  });
});
