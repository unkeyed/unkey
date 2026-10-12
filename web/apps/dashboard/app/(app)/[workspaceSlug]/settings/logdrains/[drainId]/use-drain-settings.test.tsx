import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import type { DrainDetail, DrainFormValues } from "../drain-schema";
import { useDrainSettings } from "./use-drain-settings";

const update = vi.hoisted(() => vi.fn());
vi.mock("@/lib/logdrains-query", () => ({
  useUpdateLogdrainMutation: () => ({ mutate: update }),
  useDeleteLogdrainMutation: () => ({ mutate: vi.fn() }),
}));

afterEach(() => {
  cleanup();
  update.mockClear();
});

const drain = {
  id: "drain",
  name: "Runtime",
  status: "running",
  stream: "runtime_logs",
  destination: {
    http: { url: "https://example.com/ingest", format: "ndjson", headers: ["Authorization"] },
  },
  filters: {
    severities: ["warn"],
    projectIds: ["deleted-project"],
    environmentIds: ["deleted-environment"],
  },
  batchSize: 10000,
  createdAt: 123,
} satisfies DrainDetail;

it("blocks changing one header while another retained header has no value", async () => {
  const stored = {
    ...drain,
    destination: { http: { ...drain.destination.http, headers: ["Authorization", "X-Token"] } },
  };
  const { result } = renderHook(() => useDrainSettings(stored, { onDeleted: vi.fn() }));
  act(() => result.current.form.setValue("headers.0.value", "replacement"));
  await act(() => result.current.save(vi.fn())());
  expect(update).not.toHaveBeenCalled();
  expect(result.current.form.getFieldState("headers.1.value").error?.message).toBe(
    "Enter a value for every header you keep when changing headers",
  );
});

it.each([
  {
    action: "removing one of two saved rows",
    headers: [{ name: "Authorization", value: "", stored: true }],
  },
  {
    action: "renaming a saved row",
    headers: [
      { name: "Renamed", value: "", stored: true },
      { name: "X-Token", value: "", stored: true },
    ],
  },
  {
    action: "adding a row",
    headers: [
      { name: "Authorization", value: "", stored: true },
      { name: "X-Token", value: "", stored: true },
      { name: "X-New", value: "new", stored: false },
    ],
  },
])("requires retained values when $action", async ({ headers }) => {
  const stored = {
    ...drain,
    destination: { http: { ...drain.destination.http, headers: ["Authorization", "X-Token"] } },
  };
  const { result } = renderHook(() => useDrainSettings(stored, { onDeleted: vi.fn() }));
  act(() => result.current.form.setValue("headers", headers));
  await act(() => result.current.save(vi.fn())());
  expect(update).not.toHaveBeenCalled();
  expect(result.current.form.getFieldState("headers.0.value").error?.message).toBe(
    "Enter a value for every header you keep when changing headers",
  );
});

it.each<{
  action: string;
  headers: DrainFormValues["headers"];
  expected: { name: string; value: string }[];
}>([
  {
    action: "replacing all values",
    headers: [
      { name: "Authorization", value: "first", stored: true },
      { name: "X-Token", value: "second", stored: true },
    ],
    expected: [
      { name: "Authorization", value: "first" },
      { name: "X-Token", value: "second" },
    ],
  },
  {
    action: "removing a row and replacing the retained value",
    headers: [{ name: "Authorization", value: "first", stored: true }],
    expected: [{ name: "Authorization", value: "first" }],
  },
  { action: "clearing all headers", headers: [], expected: [] },
])("sends the complete list when $action", async ({ headers, expected }) => {
  const stored = {
    ...drain,
    destination: { http: { ...drain.destination.http, headers: ["Authorization", "X-Token"] } },
  };
  const { result } = renderHook(() => useDrainSettings(stored, { onDeleted: vi.fn() }));
  act(() => result.current.form.setValue("headers", headers));
  await act(() => result.current.save(vi.fn())());
  expect(update.mock.calls[0]?.[0]).toEqual({
    logdrainId: "drain",
    destination: { http: { headers: expected } },
  });
});

it("omits unchanged saved headers when changing the destination URL", async () => {
  const { result } = renderHook(() => useDrainSettings(drain, { onDeleted: vi.fn() }));
  act(() => result.current.form.setValue("url", "https://example.com/new"));
  await act(() => result.current.save(vi.fn())());
  expect(update.mock.calls[0]?.[0]).toEqual({
    logdrainId: "drain",
    destination: { http: { url: "https://example.com/new" } },
  });
});

it("submits unrestricted runtime sources without using retained gateway selections", async () => {
  const { result } = renderHook(() => useDrainSettings(drain, { onDeleted: vi.fn() }));
  act(() => {
    result.current.form.setValue("runtimeSourceMode", "all");
    result.current.form.setValue("sourceMode", "some");
    result.current.form.setValue("projectIds", ["gateway-project"]);
  });
  await act(() => result.current.save(vi.fn())());
  expect(update.mock.calls[0]?.[0]).toEqual({
    logdrainId: "drain",
    filters: { projectIds: [], environmentIds: [] },
  });
});

it.each(["gateway_requests", "runtime_logs"] as const)(
  "preserves unavailable sources and credentials on an unrelated %s edit",
  async (stream) => {
    const stored: DrainDetail =
      stream === "gateway_requests"
        ? {
            ...drain,
            stream,
            filters: { ...drain.filters, statusClasses: ["4xx", "5xx"], severities: [] },
          }
        : drain;
    const { result } = renderHook(() => useDrainSettings(stored, { onDeleted: vi.fn() }));
    act(() => result.current.form.setValue("name", "Renamed"));
    await act(() => result.current.save(vi.fn())());
    expect(update.mock.calls[0]?.[0]).toEqual({ logdrainId: "drain", name: "Renamed" });
  },
);
