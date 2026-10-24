import {
  type FilterItemConfig,
  createOptionSearchItems,
  matchesFilterSearch,
} from "@/components/logs/checkbox/filters-popover";
import type {
  RequestLogsFilterField,
  RequestLogsFilterValue,
} from "@/lib/schemas/request-logs.filter.schema";
import { describe, expect, it, vi } from "vitest";
import {
  type RequestLogsDeploymentSearchSource,
  createRequestLogsFilterItems,
} from "./request-logs-filter-items";

const baseItems: FilterItemConfig<RequestLogsFilterField>[] = [
  { id: "status", label: "Status", component: null },
  { id: "methods", label: "Method", component: null },
  { id: "paths", label: "Path", component: null },
  { id: "deploymentId", label: "Deployment", component: null },
  { id: "host", label: "Hostname", component: null },
];

const deployments: RequestLogsDeploymentSearchSource[] = [
  { id: "dpl_123", gitBranch: "feat/requests" },
  { id: "dpl_bare", gitBranch: "" },
];

const filter = (field: RequestLogsFilterField, value: string | number): RequestLogsFilterValue => ({
  id: `${field}-${value}`,
  field,
  operator: "is",
  value,
});

const search = (query: string, current: RequestLogsFilterValue[] = []) =>
  createOptionSearchItems(
    createRequestLogsFilterItems(baseItems, deployments),
    current,
    vi.fn(),
  ).filter((item) => matchesFilterSearch(item, query));

const ids = (items: { id: string }[]) => items.map((item) => item.id);

describe("request logs filter search", () => {
  it("returns status class, method, and deployment branch or id as selectable values", () => {
    expect(ids(search("2xx"))).toEqual(["status:200"]);
    expect(ids(search("success"))).toEqual(["status:200"]);
    expect(ids(search("3xx"))).toEqual(["status:300"]);
    expect(ids(search("redirect"))).toEqual(["status:300"]);
    expect(ids(search("4xx"))).toEqual(["status:400"]);
    expect(ids(search("warning"))).toEqual(["status:400"]);
    expect(ids(search("5xx"))).toEqual(["status:500"]);
    expect(ids(search("error"))).toEqual(["status:500"]);
    expect(ids(search("get"))).toEqual(["methods:GET"]);
    expect(ids(search("PATCH"))).toEqual(["methods:PATCH"]);
    expect(ids(search("feat/requests"))).toEqual(["deploymentId:dpl_123"]);
    expect(ids(search("dpl_123"))).toEqual(["deploymentId:dpl_123"]);
    expect(ids(search("dpl_bare"))).toEqual(["deploymentId:dpl_bare"]);
  });

  it("leaves free-text filters and app selection without preset values", () => {
    const items = createRequestLogsFilterItems(
      [...baseItems, { id: "appId", label: "App", component: null }],
      deployments,
    );
    expect(items.find((item) => item.id === "paths")?.options).toBeUndefined();
    expect(items.find((item) => item.id === "host")?.options).toBeUndefined();
    expect(items.find((item) => item.id === "appId")?.options).toBeUndefined();
  });

  it("keeps exact status codes out of preset search", () => {
    expect(ids(search("5xx"))).toEqual(["status:500"]);
    expect(ids(search("error"))).toEqual(["status:500"]);
    expect(search("500")).toEqual([]);
    expect(search("418")).toEqual([]);
  });

  it("adds a method without removing other filters", () => {
    const updateFilters = vi.fn<(filters: RequestLogsFilterValue[]) => void>();
    const current = [filter("paths", "/v1/keys"), filter("methods", "POST")];
    const item = createOptionSearchItems(
      createRequestLogsFilterItems(baseItems, deployments),
      current,
      updateFilters,
    ).find((option) => option.id === "methods:GET");
    if (item?.kind !== "option") {
      throw new Error("expected GET option");
    }

    expect(item.checked).toBe(false);
    item.onSelect();

    const next = updateFilters.mock.calls[0]?.[0] ?? [];
    expect(next.map((entry) => `${entry.field}=${entry.value}`).sort()).toEqual([
      "methods=GET",
      "methods=POST",
      "paths=/v1/keys",
    ]);
    expect(next.find((entry) => entry.value === "GET")).toMatchObject({
      field: "methods",
      operator: "is",
      value: "GET",
    });
  });

  it("treats a status loaded from the URL as the same range and removes only that range", () => {
    const updateFilters = vi.fn<(filters: RequestLogsFilterValue[]) => void>();
    const current = [filter("status", "200"), filter("methods", "GET")];
    const item = createOptionSearchItems(
      createRequestLogsFilterItems(baseItems, deployments),
      current,
      updateFilters,
    ).find((option) => option.id === "status:200");
    if (item?.kind !== "option") {
      throw new Error("expected success option");
    }

    expect(item.checked).toBe(true);
    item.onSelect();

    expect(updateFilters.mock.calls[0]?.[0]).toEqual([filter("methods", "GET")]);
  });

  it("uses the deployment id as the value and as the label when there is no branch", () => {
    const updateFilters = vi.fn<(filters: RequestLogsFilterValue[]) => void>();
    const current: RequestLogsFilterValue[] = [];
    const options = createOptionSearchItems(
      createRequestLogsFilterItems(baseItems, [{ id: "dpl_bare", gitBranch: null }]),
      current,
      updateFilters,
    );
    const item = options.find((option) => option.id === "deploymentId:dpl_bare");
    if (item?.kind !== "option") {
      throw new Error("expected deployment option");
    }

    expect(item.label).toBe("dpl_bare");
    expect(item.description).toBeUndefined();
    item.onSelect();
    expect(updateFilters.mock.calls[0]?.[0]?.[0]).toMatchObject({
      field: "deploymentId",
      operator: "is",
      value: "dpl_bare",
    });
  });
});
