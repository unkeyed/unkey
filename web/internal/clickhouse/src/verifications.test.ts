import { describe, expect, it } from "vitest";
import { CapturingQuerier } from "./test-utils";
import { getHourlyKeysVerificationTimeseries } from "./verifications";

describe("getHourlyKeysVerificationTimeseries", () => {
  it("issues one grouped query for every requested key", async () => {
    const ch = new CapturingQuerier();

    await getHourlyKeysVerificationTimeseries(ch)({
      workspaceId: "ws_123",
      keyspaceId: "ks_123",
      keyIds: ["key_1", "key_2", "key_3"],
      startTime: 1_700_000_000_000,
      endTime: 1_700_100_000_000,
    });

    expect(ch.queries).toHaveLength(1);
    const query = ch.queries[0];
    expect(query).toContain("key_id as keyId");
    expect(query).toContain("AND key_id IN {keyIds: Array(String)}");
    expect(query).toContain("GROUP BY keyId, x");
    expect(query).toContain("ORDER BY keyId ASC, x ASC");
    expect(ch.params).toHaveLength(1);
    expect(ch.params[0]).toMatchObject({ keyIds: ["key_1", "key_2", "key_3"] });
  });
});
