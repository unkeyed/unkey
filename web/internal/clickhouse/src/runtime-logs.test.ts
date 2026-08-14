import { Ok, type Result } from "@unkey/error";
import { describe, expect, it } from "vitest";
import type { z } from "zod";
import type { QueryError } from "./client/error";
import type { Querier } from "./client/interface";
import { type RuntimeLogsRequest, getRuntimeLogs, runtimeLogsRequestSchema } from "./runtime-logs";

class CapturingQuerier implements Querier {
  public readonly queries: string[] = [];
  public readonly params: unknown[] = [];

  public query<TIn extends z.ZodType<unknown>, TOut extends z.ZodType<unknown>>(req: {
    query: string;
    params?: TIn;
    schema: TOut;
  }): (params: z.input<TIn>) => Promise<Result<z.output<TOut>[], QueryError>> {
    this.queries.push(req.query);
    return async (params) => {
      this.params.push(params);
      return Ok([]);
    };
  }
}

const baseRequest: RuntimeLogsRequest = {
  workspaceId: "ws_123",
  projectId: "proj_123",
  appId: [],
  deploymentId: [],
  environmentId: [],
  limit: 50,
  startTime: 1,
  endTime: 2,
  severity: null,
  region: null,
  message: null,
  attributes: null,
  attributeMatch: null,
  k8sPodNames: [],
  page: 1,
};

describe("getRuntimeLogs", () => {
  it("searches the indexed materialized attribute text", async () => {
    const ch = new CapturingQuerier();

    await getRuntimeLogs(ch)({
      ...baseRequest,
      message: "timeout",
      attributes: "tenant_id",
    });

    expect(ch.queries).toHaveLength(2);
    for (const query of ch.queries) {
      expect(query).toContain("lower(message) LIKE concat('%', lower({message: String}), '%')");
      expect(query).toContain(
        "lower(attributes_text) LIKE concat('%', lower({attributes: String}), '%')",
      );
      expect(query).not.toContain("lower(attributes),");
    }
    for (const params of ch.params) {
      expect(params).toMatchObject({
        message: "timeout",
        attributes: "tenant\\_id",
      });
    }
  });

  it("prefilters exact attribute matches before verifying the JSON path", async () => {
    const ch = new CapturingQuerier();

    await getRuntimeLogs(ch)({
      ...baseRequest,
      attributeMatch: { path: 'request.custom"id', value: "token=xyz%_" },
    });

    expect(ch.queries).toHaveLength(2);
    for (const query of ch.queries) {
      expect(query).toContain(
        "lower(attributes_text) LIKE concat('%', lower({attributeMatchPathSearch: String}), '%')",
      );
      expect(query).toContain(
        "lower(attributes_text) LIKE concat('%', lower({attributeMatchSearch: String}), '%')",
      );
      expect(query).toContain(
        "JSON_VALUE(attributes_text, {attributeMatchPath: String}) = {attributeMatchValue: String}",
      );
      expect(query).not.toContain('request.custom"id');
      expect(query).not.toContain("token=xyz");
    }
    for (const params of ch.params) {
      expect(params).toMatchObject({
        attributeMatchPath: '$."request"."custom\\"id"',
        attributeMatchPathSearch: 'custom\\\\"id',
        attributeMatchSearch: "token=xyz\\%\\_",
        attributeMatchValue: "token=xyz%_",
      });
    }
  });
});

describe("runtimeLogsRequestSchema", () => {
  it("requires trigram-sized message and attribute searches", () => {
    expect(
      runtimeLogsRequestSchema.safeParse({
        ...baseRequest,
        message: "ab",
      }).success,
    ).toBe(false);
    expect(
      runtimeLogsRequestSchema.safeParse({
        ...baseRequest,
        attributes: "ab",
      }).success,
    ).toBe(false);
    expect(
      runtimeLogsRequestSchema.safeParse({
        ...baseRequest,
        attributes: "key",
      }).success,
    ).toBe(true);
    expect(
      runtimeLogsRequestSchema.safeParse({
        ...baseRequest,
        attributeMatch: { path: "request.id", value: "xy" },
      }).success,
    ).toBe(false);
    expect(
      runtimeLogsRequestSchema.safeParse({
        ...baseRequest,
        attributeMatch: { path: "request.id", value: "xyz" },
      }).success,
    ).toBe(true);
  });
});
