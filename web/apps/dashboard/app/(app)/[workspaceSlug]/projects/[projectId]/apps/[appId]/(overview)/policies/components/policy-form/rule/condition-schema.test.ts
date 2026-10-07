import type { Policy } from "@/lib/collections/deploy/policies.schema";
import { describe, expect, it } from "vitest";
import { fromPolicy, policyFormSchema, toPolicy } from "../../../policy-kinds";
import type { PolicyFormValues } from "../../../policy-kinds";

function firewallWithRemoteIp(operator: "in" | "notIn", ranges: string): PolicyFormValues {
  return {
    type: "firewall",
    name: "office only",
    environmentId: "__all__",
    matchConditions: [{ id: "1", type: "remoteIp", operator, value: ranges }],
    action: "ACTION_DENY",
  };
}

describe("remote ip condition", () => {
  it("serializes in ranges split on commas, spaces and newlines", () => {
    const wire = toPolicy(firewallWithRemoteIp("in", "203.0.113.0/24, 198.51.100.7\n192.0.2.0/24"));
    expect(wire.match).toEqual([
      { remoteIp: { in: ["203.0.113.0/24", "198.51.100.7", "192.0.2.0/24"] } },
    ]);
  });

  it("serializes notIn ranges", () => {
    const wire = toPolicy(firewallWithRemoteIp("notIn", "198.51.100.0/24"));
    expect(wire.match).toEqual([{ remoteIp: { notIn: ["198.51.100.0/24"] } }]);
  });

  it("deserializes a stored notIn match into one comma separated line", () => {
    const stored: Policy = {
      id: "pol_1",
      name: "office only",
      enabled: true,
      type: "firewall",
      match: [{ remoteIp: { notIn: ["198.51.100.0/24", "203.0.113.7/32"] } }],
      firewall: { action: "ACTION_DENY" },
    };
    const form = fromPolicy(stored);
    expect(form.matchConditions).toMatchObject([
      { type: "remoteIp", operator: "notIn", value: "198.51.100.0/24, 203.0.113.7/32" },
    ]);
  });

  it("rejects an empty range list", () => {
    const r = policyFormSchema.safeParse(firewallWithRemoteIp("in", " , \n"));
    expect(r.success).toBe(false);
  });

  it("rejects an entry that is not an IP or CIDR and names it", () => {
    const r = policyFormSchema.safeParse(firewallWithRemoteIp("in", "203.0.113.0/24\nkebap"));
    expect(r.success).toBe(false);
    expect(r.error?.issues.map((i) => i.message)).toContain(
      "kebap is not a valid IP address or CIDR",
    );
  });

  it("accepts IPv6 entries", () => {
    const r = policyFormSchema.safeParse(
      firewallWithRemoteIp("notIn", "2001:db8::/32\n2001:db8::1"),
    );
    expect(r.success).toBe(true);
  });

  it("rejects more than 100 ranges", () => {
    const ranges = Array.from({ length: 101 }, () => "203.0.113.0/24").join("\n");
    const r = policyFormSchema.safeParse(firewallWithRemoteIp("in", ranges));
    expect(r.success).toBe(false);
  });
});
