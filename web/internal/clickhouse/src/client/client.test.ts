import { beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

const { queryMock } = vi.hoisted(() => ({ queryMock: vi.fn() }));

vi.mock("@clickhouse/client-web", () => ({
  createClient: () => ({ query: queryMock }),
}));

import { Client } from "./client";

const schema = z.object({ id: z.string() });
const params = z.object({});

describe("Client.query retry", () => {
  beforeEach(() => {
    queryMock.mockReset();
  });

  it("returns rows when the first attempt fails transiently", async () => {
    queryMock
      .mockRejectedValueOnce(new Error("request timed out"))
      .mockResolvedValueOnce({ json: async () => [{ id: "a" }] });

    const client = new Client({ url: "http://clickhouse" });
    const result = await client.query({ query: "SELECT 1", params, schema })({});

    expect(queryMock).toHaveBeenCalledTimes(2);
    expect(result.err).toBeUndefined();
    expect(result.val).toEqual([{ id: "a" }]);
  });

  it("gives up once the attempts are exhausted", async () => {
    queryMock.mockRejectedValue(new Error("request timed out"));

    const client = new Client({ url: "http://clickhouse" });
    const result = await client.query({ query: "SELECT 1", params, schema })({});

    expect(queryMock).toHaveBeenCalledTimes(2);
    expect(result.err?.message).toContain("Unable to query clickhouse");
  });

  it("does not retry rows that fail the schema", async () => {
    queryMock.mockResolvedValue({ json: async () => [{ id: 1 }] });

    const client = new Client({ url: "http://clickhouse" });
    const result = await client.query({ query: "SELECT 1", params, schema })({});

    expect(queryMock).toHaveBeenCalledTimes(1);
    expect(result.err?.message).toContain("Malformed data");
  });
});
