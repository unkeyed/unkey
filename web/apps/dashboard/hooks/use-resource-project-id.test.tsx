import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectResource } from "./use-resource-project-id";

type QueryCall = { input: Record<string, string>; options: { enabled: boolean } };
type QueryState<T> = { data?: T; isError: boolean };

let apiQuery: QueryState<{ currentApi: { projectId: string } }> = { isError: false };
let identityQuery: QueryState<{ projectId: string }> = { isError: false };
let defaultProjectQuery: { data?: { id: string } | null; isLoading: boolean; isError: boolean } = {
  isLoading: true,
  isError: false,
};
const apiCalls: QueryCall[] = [];
const identityCalls: QueryCall[] = [];

vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    api: {
      queryApiKeyDetails: {
        useQuery: (input: Record<string, string>, options: { enabled: boolean }) => {
          apiCalls.push({ input, options });
          return options.enabled ? apiQuery : { isError: false };
        },
      },
    },
    deploy: {
      project: {
        getDefault: {
          useQuery: (_input: undefined, options: { enabled: boolean }) =>
            options.enabled ? defaultProjectQuery : { isLoading: false, isError: false },
        },
      },
    },
    identity: {
      details: {
        useQuery: (input: Record<string, string>, options: { enabled: boolean }) => {
          identityCalls.push({ input, options });
          return options.enabled ? identityQuery : { isError: false };
        },
      },
    },
  },
}));

import { useResourceProjectId } from "./use-resource-project-id";

const ownerOf = (resource: ProjectResource | null) =>
  renderHook(() => useResourceProjectId(resource)).result.current;

const api: ProjectResource = { type: "api", apiId: "api_1" };
const namespace: ProjectResource = { type: "namespace", namespaceId: "ns_1" };
const identity: ProjectResource = { type: "identity", identityId: "id_1" };

beforeEach(() => {
  apiQuery = { isError: false };
  identityQuery = { isError: false };
  defaultProjectQuery = { isLoading: true, isError: false };
  apiCalls.length = 0;
  identityCalls.length = 0;
});

describe("useResourceProjectId", () => {
  it("returns null with no resource and runs no lookup", () => {
    expect(ownerOf(null)).toBeNull();
    expect(apiCalls.map((call) => call.options.enabled)).toEqual([false]);
    expect(identityCalls.map((call) => call.options.enabled)).toEqual([false]);
  });

  it("resolves the owning project of a keyspace", () => {
    apiQuery = { data: { currentApi: { projectId: "proj_1" } }, isError: false };
    expect(ownerOf(api)).toEqual({ state: "resolved", projectId: "proj_1" });
    expect(apiCalls).toEqual([{ input: { apiId: "api_1" }, options: { enabled: true } }]);
  });

  it("resolves a namespace to the workspace default project", () => {
    defaultProjectQuery = { data: { id: "proj_2" }, isLoading: false, isError: false };
    expect(ownerOf(namespace)).toEqual({ state: "resolved", projectId: "proj_2" });
  });

  it("resolves the owning project of an identity", () => {
    identityQuery = { data: { projectId: "proj_3" }, isError: false };
    expect(ownerOf(identity)).toEqual({ state: "resolved", projectId: "proj_3" });
    expect(identityCalls).toEqual([{ input: { identityId: "id_1" }, options: { enabled: true } }]);
  });

  it("is loading until the lookup resolves", () => {
    expect(ownerOf(api)).toEqual({ state: "loading" });
    expect(ownerOf(namespace)).toEqual({ state: "loading" });
    expect(ownerOf(identity)).toEqual({ state: "loading" });
  });

  it("is unknown when the lookup fails", () => {
    apiQuery = { isError: true };
    identityQuery = { isError: true };
    defaultProjectQuery = { isLoading: false, isError: true };
    expect(ownerOf(api)).toEqual({ state: "unknown" });
    expect(ownerOf(namespace)).toEqual({ state: "unknown" });
    expect(ownerOf(identity)).toEqual({ state: "unknown" });
  });

  it("is unknown when the workspace has no default project", () => {
    defaultProjectQuery = { data: null, isLoading: false, isError: false };
    expect(ownerOf(namespace)).toEqual({ state: "unknown" });
  });
});
