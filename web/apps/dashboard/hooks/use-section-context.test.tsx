import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let params: Record<string, string> = {};
let segments: string[] = [];

vi.mock("next/navigation", () => ({
  useParams: () => params,
  useSelectedLayoutSegments: () => segments,
}));

import { useSectionContext } from "./use-section-context";

const context = () => renderHook(() => useSectionContext()).result.current;

beforeEach(() => {
  params = {};
  segments = [];
});

describe("outside a project", () => {
  it.each([
    [{ apiId: "api_1" }, { type: "api", apiId: "api_1", projectId: undefined }],
    [{ namespaceId: "ns_1" }, { type: "namespace", namespaceId: "ns_1", projectId: undefined }],
    [{ identityId: "id_1" }, { type: "identity", identityId: "id_1", projectId: undefined }],
  ])("reads %o as a resource", (routeParams, expected) => {
    params = routeParams;
    expect(context()).toEqual(expected);
  });

  it("reads the settings and authorization segments", () => {
    segments = ["(app)", "settings"];
    expect(context()).toEqual({ type: "settings" });
    segments = ["(app)", "authorization"];
    expect(context()).toEqual({ type: "authorization" });
  });

  it("falls back to the workspace", () => {
    expect(context()).toEqual({ type: "workspace" });
  });
});

describe("inside a project", () => {
  it("keeps a bare project url on the project", () => {
    params = { projectId: "proj_1" };
    expect(context()).toEqual({ type: "project", projectId: "proj_1", appId: undefined });
  });

  it("keeps an app url on the project", () => {
    params = { projectId: "proj_1", appId: "app_1" };
    expect(context()).toEqual({ type: "project", projectId: "proj_1", appId: "app_1" });
  });

  it.each([
    [{ apiId: "api_1" }, { type: "api", apiId: "api_1", projectId: "proj_1" }],
    [{ namespaceId: "ns_1" }, { type: "namespace", namespaceId: "ns_1", projectId: "proj_1" }],
    [{ identityId: "id_1" }, { type: "identity", identityId: "id_1", projectId: "proj_1" }],
  ])("lets %o win over the project id and carries it", (routeParams, expected) => {
    params = { projectId: "proj_1", ...routeParams };
    expect(context()).toEqual(expected);
  });
});
