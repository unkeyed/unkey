import { describe, expect, it } from "vitest";
import { CapturingQuerier } from "./test-utils";
import { getIdentityLogs, getKeyDetailsLogs } from "./verifications";

const identityRequest = {
  workspaceId: "ws_123",
  keyIds: ["key_123"],
  limit: 50,
  startTime: 1,
  endTime: 2,
  tags: null,
  outcomes: null,
};

describe("getIdentityLogs", () => {
  it("pages past rows that share the cursor's millisecond", async () => {
    const ch = new CapturingQuerier();

    await getIdentityLogs(ch)({
      ...identityRequest,
      cursorTime: 1_700_000_000_000,
      cursorRequestId: "req_b",
    });

    expect(ch.queries[0]).toContain(
      "AND (time, request_id) < ({cursorTime: UInt64}, {cursorRequestId: String})",
    );
    expect(ch.queries[0]).toContain("ORDER BY time DESC, request_id DESC");
    expect(ch.params[0]).toMatchObject({
      cursorTime: 1_700_000_000_000,
      cursorRequestId: "req_b",
    });
  });

  it("reads the first page without a cursor condition", async () => {
    const ch = new CapturingQuerier();

    await getIdentityLogs(ch)({ ...identityRequest, cursorTime: null, cursorRequestId: null });

    expect(ch.queries[0]).not.toContain("(time, request_id) <");
  });
});

describe("getKeyDetailsLogs", () => {
  it("orders rows that share a millisecond by request id", async () => {
    const ch = new CapturingQuerier();

    await getKeyDetailsLogs(ch)({
      workspaceId: "ws_123",
      keyspaceId: "ks_123",
      keyId: "key_123",
      limit: 50,
      startTime: 1,
      endTime: 2,
      tags: null,
      outcomes: null,
      cursorTime: null,
      offset: 50,
    });

    expect(ch.queries[0]).toContain("ORDER BY time DESC, request_id DESC");
  });
});
