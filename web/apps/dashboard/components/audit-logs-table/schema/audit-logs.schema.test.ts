import { describe, expect, it } from "vitest";
import { DEFAULT_BUCKET_NAME, auditLogsQueryPayload } from "./audit-logs.schema";

const base = { since: "", events: null, users: null, rootKeys: null };

describe("auditLogsQueryPayload bucket", () => {
  it("defaults to the dashboard bucket", () => {
    const parsed = auditLogsQueryPayload.parse(base);
    expect(parsed.bucket).toBe(DEFAULT_BUCKET_NAME);
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
