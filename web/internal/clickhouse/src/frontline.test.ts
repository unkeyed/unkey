import { describe, expect, it } from "vitest";
import { getRequestLogs, type RequestLogsRequest, requestLogsRequestSchema } from "./frontline";
import { CapturingQuerier } from "./test-utils";

const baseRequest: RequestLogsRequest = {
  workspaceId: "ws_123",
  projectId: "proj_123",
  appId: [],
  deploymentId: [],
  environmentId: [],
  limit: 50,
  startTime: 1,
  endTime: 2,
  since: null,
  statusCodes: null,
  methods: null,
  paths: null,
  host: null,
  requestId: null,
  region: null,
  page: 1,
};

describe("getRequestLogs", () => {
  it("adds direct indexed predicates only for active filters", async () => {
    const ch = new CapturingQuerier();

    await getRequestLogs(ch)({
      ...baseRequest,
      appId: ["app_123"],
      deploymentId: ["dpl_123"],
      environmentId: ["env_123"],
      host: ["api.example.com"],
      requestId: ["req_123"],
      region: ["us-east-1"],
      paths: [
        { operator: "is", value: "/health" },
        { operator: "startsWith", value: "/api" },
        { operator: "contains", value: "%users_" },
      ],
    });

    expect(ch.queries).toHaveLength(2);
    for (const query of ch.queries) {
      expect(query).toContain("app_id IN {appId: Array(String)}");
      expect(query).toContain("deployment_id IN {deploymentId: Array(String)}");
      expect(query).toContain("environment_id IN {environmentId: Array(String)}");
      expect(query).toContain("host IN {host: Array(String)}");
      expect(query).toContain("request_id IN {requestId: Array(String)}");
      expect(query).toContain("region IN {region: Array(String)}");
      expect(query).toContain("path = {pathValue0: String}");
      expect(query).toContain("startsWith(path, {pathValue1: String})");
      expect(query).toContain("path LIKE concat('%', {pathValue2: String}, '%')");
    }
    for (const params of ch.params) {
      expect(params).toMatchObject({
        pathValue2: "\\%users\\_",
      });
    }
  });

  it("omits inactive optional predicates", async () => {
    const ch = new CapturingQuerier();

    await getRequestLogs(ch)(baseRequest);

    expect(ch.queries).toHaveLength(2);
    for (const query of ch.queries) {
      expect(query).not.toContain("host IN");
      expect(query).not.toContain("request_id IN");
      expect(query).not.toContain("region IN");
      expect(query).not.toContain("path LIKE");
      expect(query).not.toContain("startsWith(path");
    }
  });
});

describe("requestLogsRequestSchema", () => {
  it.each([
    [{ operator: "contains", value: "ab" }, false],
    [{ operator: "is", value: "/" }, true],
  ] as const)("validates indexed path bounds", (path, valid) => {
    expect(requestLogsRequestSchema.safeParse({ ...baseRequest, paths: [path] }).success).toBe(
      valid,
    );
  });
});
