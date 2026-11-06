import { AUDIT_LOG_BUCKET_UNKEY_MUTATIONS } from "@unkey/schema/src/auditlog";
import { describe, expect, it } from "vitest";
import { auditLogsQueryPayload } from "./audit-logs.schema";

const base = { since: "", events: null, users: null, rootKeys: null };

describe("auditLogsQueryPayload bucket", () => {
  it("defaults to the dashboard bucket", () => {
    const parsed = auditLogsQueryPayload.parse(base);
    expect(parsed.bucket).toBe(AUDIT_LOG_BUCKET_UNKEY_MUTATIONS);
  });

  it("accepts the dashboard bucket", () => {
    const parsed = auditLogsQueryPayload.parse({ ...base, bucket: "unkey_mutations" });
    expect(parsed.bucket).toBe("unkey_mutations");
  });

  it("rejects the back office bucket", () => {
    expect(auditLogsQueryPayload.safeParse({ ...base, bucket: "unkey_backoffice" }).success).toBe(
      false,
    );
  });

  it("rejects arbitrary bucket names", () => {
    expect(auditLogsQueryPayload.safeParse({ ...base, bucket: "audit_xyz" }).success).toBe(false);
  });
});

describe("auditLogsQueryPayload since", () => {
  it("accepts an empty since", () => {
    expect(auditLogsQueryPayload.safeParse(base).success).toBe(true);
  });

  it("accepts a relative duration", () => {
    expect(auditLogsQueryPayload.safeParse({ ...base, since: "1w2d3h" }).success).toBe(true);
  });

  it("rejects a malformed since instead of failing later", () => {
    expect(auditLogsQueryPayload.safeParse({ ...base, since: "abc" }).success).toBe(false);
    expect(auditLogsQueryPayload.safeParse({ ...base, since: "-1h" }).success).toBe(false);
    expect(auditLogsQueryPayload.safeParse({ ...base, since: "1h!" }).success).toBe(false);
  });
});
