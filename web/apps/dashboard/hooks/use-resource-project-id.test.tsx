import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectResource } from "./use-resource-project-id";

type QueryCall = { input: Record<string, string>; options: { enabled: boolean } };
type QueryState<T> = { data?: T; isError: boolean };

let apiQuery: QueryState<{ currentApi: { projectId: string } }> = { isError: false };
let identityQuery: QueryState<{ projectId: string }> = { isError: false };
let namespaces: {
  rows: { id: string; projectId: string }[];
  isLoading: boolean;
  isError: boolean;
} = { rows: [], isLoading: true, isError: false };
const apiCalls: QueryCall[] = [];
const identityCalls: QueryCall[] = [];

type Row = { id: string; projectId: string };
type Stage = {
  rows: Row[];
  where: (predicate: (row: { namespace: Row }) => boolean) => Stage;
};

function stage(current: Row[]): Stage {
  return {
    rows: current,
    where: (predicate) => stage(current.filter((row) => predicate({ namespace: row }))),
  };
}

vi.mock("@/lib/collections", () => ({ collection: { ratelimitNamespaces: {} } }));
vi.mock("@tanstack/react-db", () => ({
  eq: (left: unknown, right: unknown) => left === right,
  useLiveQuery: (build: (q: { from: () => Stage }) => Stage | null) => {
    const built = build({ from: () => stage(namespaces.rows) });
    return { data: built?.rows, isLoading: namespaces.isLoading, isError: namespaces.isError };
  },
}));
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
  namespaces = { rows: [], isLoading: true, isError: false };
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

  it("resolves the owning project of a namespace", () => {
    namespaces = {
      rows: [
        { id: "ns_other", projectId: "proj_other" },
        { id: "ns_1", projectId: "proj_2" },
      ],
      isLoading: false,
      isError: false,
    };
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
    namespaces = { rows: [], isLoading: false, isError: true };
    expect(ownerOf(api)).toEqual({ state: "unknown" });
    expect(ownerOf(namespace)).toEqual({ state: "unknown" });
    expect(ownerOf(identity)).toEqual({ state: "unknown" });
  });

  it("is unknown when the namespace is not in the collection", () => {
    namespaces = {
      rows: [{ id: "ns_other", projectId: "proj_other" }],
      isLoading: false,
      isError: false,
    };
    expect(ownerOf(namespace)).toEqual({ state: "unknown" });
  });
});
