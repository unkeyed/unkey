import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import type { DrainDetail } from "../drain-schema";
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
  committedOffsetInsertedAt: 0,
  consecutiveFailures: 0,
} satisfies DrainDetail;

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
        ? { ...drain, stream, filters: { ...drain.filters, statusClasses: [4, 5], severities: [] } }
        : drain;
    const { result } = renderHook(() => useDrainSettings(stored, { onDeleted: vi.fn() }));
    act(() => result.current.form.setValue("name", "Renamed"));
    await act(() => result.current.save(vi.fn())());
    expect(update.mock.calls[0]?.[0]).toEqual({ logdrainId: "drain", name: "Renamed" });
  },
);
