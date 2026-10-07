import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

const { queryMock } = vi.hoisted(() => ({ queryMock: vi.fn() }));

vi.mock("@clickhouse/client-web", () => ({
  createClient: () => ({ query: queryMock, insert: vi.fn() }),
}));

import { Client } from "./client";

const schema = z.object({ y: z.number() });

const select = () => new Client({ url: "http://clickhouse" }).query({ query: "SELECT 1", schema });

describe("Client.query retry", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    queryMock.mockReset();
  });

  it("retries a fast transport failure and returns the successful attempt", async () => {
    queryMock
      .mockRejectedValueOnce(new Error("fetch failed"))
      .mockResolvedValueOnce({ json: async () => [{ y: 1 }] });

    const result = await select()({});

    expect(result.err).toBeUndefined();
    expect(result.val).toEqual([{ y: 1 }]);
    expect(queryMock).toHaveBeenCalledTimes(2);
  });

  it("returns a QueryError after the bounded attempts", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    queryMock.mockRejectedValue(new Error("fetch failed"));

    const result = await select()({});

    expect(result.err?.message).toContain("Unable to query clickhouse: fetch failed");
    expect(queryMock).toHaveBeenCalledTimes(3);
  });

  it("does not retry an attempt that consumed the slow-attempt budget", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    vi.spyOn(Date, "now").mockReturnValueOnce(0).mockReturnValueOnce(10_000);
    queryMock.mockRejectedValue(new Error("The operation was aborted due to timeout"));

    const result = await select()({});

    expect(result.err).toBeDefined();
    expect(queryMock).toHaveBeenCalledTimes(1);
  });

  it("does not retry malformed rows", async () => {
    queryMock.mockResolvedValueOnce({ json: async () => [{ y: "not-a-number" }] });

    const result = await select()({});

    expect(result.err?.message).toContain("Malformed data");
    expect(queryMock).toHaveBeenCalledTimes(1);
  });
});
