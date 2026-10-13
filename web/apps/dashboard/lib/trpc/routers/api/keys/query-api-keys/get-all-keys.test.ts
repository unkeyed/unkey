import { newId } from "@unkey/id";
import { expect, it, vi } from "vitest";
import { getAllKeys } from "./get-all-keys";

const mocks = vi.hoisted(() => ({ findKeys: vi.fn(), findKeyspace: vi.fn() }));

vi.mock("@/lib/db", async () => {
  const real = await import("@unkey/db");
  return {
    ...real,
    db: {
      query: {
        keyAuth: { findFirst: mocks.findKeyspace },
        keys: { findMany: mocks.findKeys },
      },
    },
  };
});
vi.mock("@/lib/clickhouse", () => ({ clickhouse: {} }));

it.each([0, 1, 2, 3])(
  "returns a bounded page and continuation for %i fetched keys",
  async (size) => {
    const keyspaceId = newId("keyAuth");
    const rows = Array.from({ length: size }, (_, index) => ({
      pk: index + 1,
      id: newId("key"),
      name: `Key ${index}`,
      identityId: null,
      enabled: true,
      expires: null,
      identity: null,
      updatedAtM: null,
      prefix: "test",
      start: `start-${index}`,
      end: "end",
      lastUsedAt: index * 1000,
      meta: null,
      remaining: null,
      refillAmount: null,
      refillDay: null,
      ratelimits: [],
    }));
    mocks.findKeyspace.mockResolvedValue({ id: keyspaceId });
    mocks.findKeys.mockResolvedValue(rows);

    const result = await getAllKeys({ keyspaceId, workspaceId: newId("workspace"), limit: 2 });

    expect(result.keys.map((key) => key.id)).toEqual(rows.slice(0, 2).map((key) => key.id));
    expect(result.nextCursor).toEqual(
      size > 2 ? { pk: 2, id: rows[1].id, start: "start-1", lastUsedAt: 1000 } : null,
    );
  },
);
