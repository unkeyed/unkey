import { describe, expect, it } from "vitest";
import { getDeployUsageTimeseries } from "./deploy_billing";
import { CapturingQuerier } from "./test-utils";

describe("getDeployUsageTimeseries", () => {
  it("aggregates all instances and applies scope filters", async () => {
    const ch = new CapturingQuerier();

    await getDeployUsageTimeseries(ch)({
      workspaceId: "ws_123",
      periodStart: 1,
      end: 2,
      interval: "hour",
      groupBy: "app",
      projectId: "project_123",
      appIds: ["app_123"],
      environmentIds: ["env_123"],
    });

    expect(ch.queries).toHaveLength(1);
    expect(ch.queries[0]).toContain("sum(network_egress_public_bytes)");
    expect(ch.queries[0]).toContain("GROUP BY bucket, groupId");
    expect(ch.queries[0]).not.toContain("container_uid");
    expect(ch.params[0]).toEqual({
      workspaceId: "ws_123",
      periodStart: 1,
      end: 2,
      interval: "hour",
      groupBy: "app",
      projectId: "project_123",
      appIds: ["app_123"],
      environmentIds: ["env_123"],
    });
    expect(ch.queries[0]).toContain("OR app_id IN {appIds: Array(String)}");
    expect(ch.queries[0]).toContain("OR environment_id IN {environmentIds: Array(String)}");
  });
});
