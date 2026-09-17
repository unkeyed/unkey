import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectResource } from "./use-resource-project-id";

type QueryCall = { input: Record<string, string>; options: { enabled: boolean } };

let apiData: { currentApi: { projectId: string | null } } | undefined;
let identityData: { projectId: string } | undefined;
let namespaceRows: { id: string; projectId: string | null }[] = [];
const apiCalls: QueryCall[] = [];
const identityCalls: QueryCall[] = [];

type Stage = {
  rows: typeof namespaceRows;
  where: (predicate: (row: { namespace: (typeof namespaceRows)[number] }) => boolean) => Stage;
};

function stage(current: typeof namespaceRows): Stage {
  return {
    rows: current,
    where: (predicate) => stage(current.filter((row) => predicate({ namespace: row }))),
  };
}

vi.mock("@/lib/collections", () => ({ collection: { ratelimitNamespaces: {} } }));
vi.mock("@tanstack/react-db", () => ({
  eq: (left: unknown, right: unknown) => left === right,
  useLiveQuery: (build: (q: { from: () => Stage }) => Stage | null) => {
    const built = build({ from: () => stage(namespaceRows) });
    return { data: built?.rows };
  },
}));
vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    api: {
      queryApiKeyDetails: {
        useQuery: (input: Record<string, string>, options: { enabled: boolean }) => {
          apiCalls.push({ input, options });
          return { data: options.enabled ? apiData : undefined };
        },
      },
    },
    identity: {
      details: {
        useQuery: (input: Record<string, string>, options: { enabled: boolean }) => {
          identityCalls.push({ input, options });
          return { data: options.enabled ? identityData : undefined };
        },
      },
    },
  },
}));

import { useResourceProjectId } from "./use-resource-project-id";

const ownerOf = (resource: ProjectResource | null) =>
  renderHook(() => useResourceProjectId(resource)).result.current;

beforeEach(() => {
  apiData = undefined;
  identityData = undefined;
  namespaceRows = [];
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
    apiData = { currentApi: { projectId: "proj_1" } };
    expect(ownerOf({ type: "api", apiId: "api_1" })).toBe("proj_1");
    expect(apiCalls).toEqual([{ input: { apiId: "api_1" }, options: { enabled: true } }]);
  });

  it("resolves the owning project of a namespace", () => {
    namespaceRows = [
      { id: "ns_other", projectId: "proj_other" },
      { id: "ns_1", projectId: "proj_2" },
    ];
    expect(ownerOf({ type: "namespace", namespaceId: "ns_1" })).toBe("proj_2");
  });

  it("resolves the owning project of an identity", () => {
    identityData = { projectId: "proj_3" };
    expect(ownerOf({ type: "identity", identityId: "id_1" })).toBe("proj_3");
    expect(identityCalls).toEqual([{ input: { identityId: "id_1" }, options: { enabled: true } }]);
  });

  it("returns null until the query resolves", () => {
    expect(ownerOf({ type: "api", apiId: "api_1" })).toBeNull();
    expect(ownerOf({ type: "namespace", namespaceId: "ns_1" })).toBeNull();
    expect(ownerOf({ type: "identity", identityId: "id_1" })).toBeNull();
  });

  it("returns null when the resource belongs to no project", () => {
    apiData = { currentApi: { projectId: null } };
    expect(ownerOf({ type: "api", apiId: "api_1" })).toBeNull();
  });
});
