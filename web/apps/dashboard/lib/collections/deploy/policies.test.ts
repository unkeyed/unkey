import { afterEach, describe, expect, it, vi } from "vitest";
import { type PolicyRow, replacePolicyLists, rowKey } from "./policies";
import { type Policy, policyIdentity } from "./policies.schema";

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

describe("policyIdentity", () => {
  it("separates two types that share a name", () => {
    expect(policyIdentity("firewall", "Guard")).not.toBe(policyIdentity("ratelimit", "Guard"));
  });

  it("folds case and surrounding space", () => {
    expect(policyIdentity("firewall", "  Guard ")).toBe(policyIdentity("firewall", "guard"));
  });
});
