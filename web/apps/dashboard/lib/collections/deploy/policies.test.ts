import { toast } from "@unkey/ui";
import { afterEach, describe, expect, it, vi } from "vitest";
import { queryClient } from "../client";
import { type PolicyRow, replacePolicyLists, rowKey, writePolicies } from "./policies";
import { type Policy, fromWirePolicy, policyMatchKey } from "./policies.schema";

const LABELS = { loading: "Saving...", success: "Saved", error: "Failed" };

function firewallPolicy(id: string, name: string, match?: Policy["match"]): Policy {
  return {
    id,
    name,
    enabled: true,
    type: "firewall",
    firewall: { action: "ACTION_DENY" },
    ...(match !== undefined ? { match } : {}),
  };
}

function firewallRow(id: string, name: string, environmentId: string, order: number): PolicyRow {
  return {
    ...firewallPolicy(id, name),
    environmentId,
    projectId: "proj_KEBAP",
    appId: "app_KEBAP",
    _order: order,
  };
}

function captureRequests(): { url: string; body: Record<string, unknown> }[] {
  const requests: { url: string; body: Record<string, unknown> }[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const request = input instanceof Request ? input : new Request(input, init);
      requests.push({ url: request.url, body: await request.clone().json() });
      return new Response(JSON.stringify({ meta: { requestId: "req_1" }, data: {} }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }),
  );
  return requests;
}

describe("rowKey", () => {
  it("scopes a policy id to its environment, so two copies do not collide", () => {
    expect(rowKey("env_prod", "pol_1")).not.toBe(rowKey("env_preview", "pol_1"));
  });
});

// A full replace: the list must hold every policy of the environment, in the
// order it is meant to be evaluated, also when two policies share a name.
describe("replacePolicyLists", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends every policy, in the given order, when two share a name", async () => {
    const requests = captureRequests();

    await replacePolicyLists(
      [
        {
          environmentId: "env_1",
          projectId: "proj_KEBAP",
          appId: "app_KEBAP",
          policies: [
            firewallRow("pol_3", "Auth", "env_1", 0),
            firewallRow("pol_1", "Ratelimit", "env_1", 1),
            firewallRow("pol_2", "Ratelimit", "env_1", 2),
          ],
        },
      ],
      LABELS,
    );

    expect(requests).toHaveLength(1);
    expect(requests[0].url).toBe("http://localhost:3000/proxy/v2/gateway.setPolicies");
    const sent = requests[0].body.policies as { name: string }[];
    expect(sent.map((p) => p.name)).toEqual(["Auth", "Ratelimit", "Ratelimit"]);
  });

  it("sends one request per environment", async () => {
    const requests = captureRequests();

    await replacePolicyLists(
      [
        {
          environmentId: "env_prod",
          projectId: "proj_KEBAP",
          appId: "app_KEBAP",
          policies: [firewallRow("pol_1", "A", "env_prod", 0)],
        },
        {
          environmentId: "env_preview",
          projectId: "proj_KEBAP",
          appId: "app_KEBAP",
          policies: [firewallRow("pol_2", "A", "env_preview", 0)],
        },
      ],
      LABELS,
    );

    expect(requests).toHaveLength(2);
  });

  it("sends an empty list, so deleting the last policy clears the environment", async () => {
    const requests = captureRequests();

    await replacePolicyLists(
      [{ environmentId: "env_1", projectId: "proj_KEBAP", appId: "app_KEBAP", policies: [] }],
      LABELS,
    );

    expect(requests).toHaveLength(1);
    expect(requests[0].body.policies).toEqual([]);
  });

  // `save` calls this with the environments that need an append, which is often
  // none of them. That must not fire a request or raise a toast.
  it("does nothing when there is nothing to replace", async () => {
    const requests = captureRequests();

    await replacePolicyLists([], LABELS);

    expect(requests).toHaveLength(0);
  });
});

describe("writePolicies", () => {
  const scope = {
    projectId: "proj_KEBAP",
    appId: "app_KEBAP",
    environments: { production: "env_prod", preview: "env_prev" },
  };
  const clearProduction = () => ({ type: "write" as const, lists: { production: [] } });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    queryClient.clear();
  });

  it("refuses to write before both environments are loaded", async () => {
    const requests = captureRequests();
    const error = vi.spyOn(toast, "error");
    queryClient.setQueryData(["policies", "env_prod"], [firewallRow("pol_1", "A", "env_prod", 0)]);

    expect(await writePolicies(scope, clearProduction, LABELS)).toBe(false);
    expect(requests).toHaveLength(0);
    expect(error).toHaveBeenCalledWith(
      "Couldn't find the environment. Reload the page and try again.",
    );
  });

  it("rejects the next edit when the refetch after a write failed", async () => {
    const requests = captureRequests();
    const error = vi.spyOn(toast, "error");
    // Cache entries with no queryFn, so the refetch after the write fails.
    queryClient.setQueryDefaults(["policies"], { retry: false });
    queryClient.setQueryData(["policies", "env_prod"], [firewallRow("pol_1", "A", "env_prod", 0)]);
    queryClient.setQueryData(["policies", "env_prev"], []);

    const first = writePolicies(scope, clearProduction, LABELS);
    const second = writePolicies(scope, clearProduction, LABELS);

    expect(await first).toBe(true);
    expect(await second).toBe(false);
    expect(requests).toHaveLength(1);
    expect(error).toHaveBeenCalledWith("Policies are out of date. Reload and try again.");
  });
});

describe("policyMatchKey", () => {
  it("separates two types that share a name", () => {
    expect(policyMatchKey("firewall", "Guard")).not.toBe(policyMatchKey("ratelimit", "Guard"));
  });

  it("folds surrounding space but not case", () => {
    expect(policyMatchKey("firewall", "  Guard ")).toBe(policyMatchKey("firewall", "Guard"));
    expect(policyMatchKey("firewall", "Guard")).not.toBe(policyMatchKey("firewall", "guard"));
  });
});

describe("fromWirePolicy remoteIp match", () => {
  const wire = (remoteIp: unknown) => ({
    id: "pol_1",
    name: "office only",
    enabled: true,
    firewall: { action: "ACTION_DENY" },
    match: [{ remoteIp }],
  });

  it("accepts in and notIn lists", () => {
    expect(fromWirePolicy(wire({ in: ["203.0.113.0/24"] })).match).toEqual([
      { remoteIp: { in: ["203.0.113.0/24"] } },
    ]);
    expect(fromWirePolicy(wire({ notIn: ["198.51.100.7/32"] })).match).toEqual([
      { remoteIp: { notIn: ["198.51.100.7/32"] } },
    ]);
  });

  it("rejects both lists, neither list, and an empty list", () => {
    expect(() =>
      fromWirePolicy(wire({ in: ["203.0.113.0/24"], notIn: ["198.51.100.0/24"] })),
    ).toThrow();
    expect(() => fromWirePolicy(wire({}))).toThrow();
    expect(() => fromWirePolicy(wire({ in: [] }))).toThrow();
  });
});

describe("fromWirePolicy variant", () => {
  it("names the policy type after the variant field it carries", () => {
    expect(fromWirePolicy({ id: "p", name: "Spec", enabled: true, openapi: {} }).type).toBe(
      "openapi",
    );
  });

  it("rejects a policy with no known variant field", () => {
    expect(() => fromWirePolicy({ id: "p", name: "x", enabled: true, waf: {} })).toThrow(
      "unknown gateway policy variant",
    );
  });
});
