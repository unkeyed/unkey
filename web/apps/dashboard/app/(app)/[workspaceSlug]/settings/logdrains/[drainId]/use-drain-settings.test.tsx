import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import type { DrainDetail } from "../drain-schema";
import { useDrainSettings } from "./use-drain-settings";

const update = vi.hoisted(() => vi.fn());
vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    useUtils: () => ({ logdrain: { list: { invalidate: vi.fn() }, get: { invalidate: vi.fn() } } }),
    logdrain: {
      update: { useMutation: () => ({ mutate: update }) },
      delete: { useMutation: () => ({ mutate: vi.fn() }) },
    },
  },
}));

afterEach(() => {
  cleanup();
  update.mockClear();
});

const drain = {
  id: "drain",
  name: "Runtime",
  status: "running",
  kind: "http",
  stream: "runtime_logs",
  config: { url: "https://example.com/ingest", format: "ndjson", headers: ["Authorization"] },
  eventTypes: [],
  outcomes: [],
  keySpaceIds: [],
  statusClasses: [],
  severities: ["warn"],
  namespaceIds: [],
  passed: [],
  projectIds: ["deleted-project"],
  appIds: [],
  environmentIds: ["deleted-environment"],
} satisfies DrainDetail;

it("submits unrestricted runtime sources without using retained gateway selections", async () => {
  const { result } = renderHook(() => useDrainSettings(drain, { onDeleted: vi.fn() }));
  act(() => {
    result.current.form.setValue("runtimeSourceMode", "all");
    result.current.form.setValue("sourceMode", "some");
    result.current.form.setValue("projectIds", ["gateway-project"]);
  });
  await act(() => result.current.save(vi.fn())());
  expect(update.mock.calls[0]?.[0]).toEqual({ id: "drain", projectIds: [], environmentIds: [] });
});

it.each(["gateway_requests", "runtime_logs"] as const)(
  "preserves unavailable sources and credentials on an unrelated %s edit",
  async (stream) => {
    const stored: DrainDetail =
      stream === "gateway_requests"
        ? { ...drain, stream, statusClasses: [4, 5], severities: [] }
        : drain;
    const { result } = renderHook(() => useDrainSettings(stored, { onDeleted: vi.fn() }));
    act(() => result.current.form.setValue("name", "Renamed"));
    await act(() => result.current.save(vi.fn())());
    expect(update.mock.calls[0]?.[0]).toEqual({ id: "drain", name: "Renamed" });
  },
);
