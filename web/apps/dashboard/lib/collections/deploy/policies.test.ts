import { afterEach, describe, expect, it, vi } from "vitest";
import {
  type PolicyChange,
  type PolicyRow,
  dispatchUpdate,
  environmentsInMutations,
  orderedPolicies,
  policiesAfterMutations,
  reorderPolicies,
  rowKey,
} from "./policies";
import type { Policy } from "./policies.schema";

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

function stateOf(...rows: PolicyRow[]): Map<string, PolicyRow> {
  return new Map(rows.map((r) => [rowKey(r.environmentId, r.id), r]));
}

function changeOf(row: PolicyRow, removed = false): PolicyChange {
  return { key: rowKey(row.environmentId, row.id), row, removed };
}

// A collection handler runs before TanStack DB recomputes optimistic state,
// so `policies.state` still holds the rows from before the mutation. A full
// replace built from state alone omits the change.
describe("policiesAfterMutations", () => {
  it("includes an inserted policy that collection state does not know about yet", () => {
    const inserted = firewallRow("pol_new", "KEBAP", "env_1", 0);

    const result = policiesAfterMutations(stateOf(), "env_1", [changeOf(inserted)]);

    expect(result.map((p) => p.name)).toEqual(["KEBAP"]);
  });

  it("appends an insert to an environment that already has policies", () => {
    const existing = firewallRow("pol_1", "First", "env_1", 0);
    const inserted = firewallRow("pol_new", "Second", "env_1", 1);

    const result = policiesAfterMutations(stateOf(existing), "env_1", [changeOf(inserted)]);

    expect(result.map((p) => p.name)).toEqual(["First", "Second"]);
  });

  it("drops a deleted policy that collection state still holds", () => {
    const kept = firewallRow("pol_1", "Kept", "env_1", 0);
    const removed = firewallRow("pol_2", "Removed", "env_1", 1);

    const result = policiesAfterMutations(stateOf(kept, removed), "env_1", [
      changeOf(removed, true),
    ]);

    expect(result.map((p) => p.name)).toEqual(["Kept"]);
  });

  it("yields an empty list only when the last policy is genuinely deleted", () => {
    const only = firewallRow("pol_1", "Only", "env_1", 0);

    const result = policiesAfterMutations(stateOf(only), "env_1", [changeOf(only, true)]);

    expect(result).toEqual([]);
  });

  it("ignores mutations that belong to a different environment", () => {
    const prod = firewallRow("pol_1", "Prod", "env_prod", 0);
    const previewInsert = firewallRow("pol_2", "Preview", "env_preview", 0);

    const result = policiesAfterMutations(stateOf(prod), "env_prod", [changeOf(previewInsert)]);

    expect(result.map((p) => p.name)).toEqual(["Prod"]);
  });
});

describe("orderedPolicies", () => {
  it("sorts rows into evaluation order and strips row-only fields", () => {
    const rows = [
      firewallRow("pol_b", "Second", "env_1", 1),
      firewallRow("pol_a", "First", "env_1", 0),
    ];

    const result = orderedPolicies(rows);

    expect(result.map((p) => p.name)).toEqual(["First", "Second"]);
    for (const p of result) {
      // Row-only fields ride along; the SDK's outbound schema drops them.
      expect(p).toHaveProperty("id");
      expect(p).toHaveProperty("type", "firewall");
    }
  });

  it("does not mutate the input array's order", () => {
    const rows = [
      firewallRow("pol_b", "Second", "env_1", 1),
      firewallRow("pol_a", "First", "env_1", 0),
    ];
    orderedPolicies(rows);
    expect(rows.map((r) => r.name)).toEqual(["Second", "First"]);
  });
});

describe("environmentsInMutations", () => {
  it("dedupes to one entry per environment", () => {
    const rows = [
      firewallRow("pol_1", "Prod copy", "env_prod", 0),
      firewallRow("pol_1", "Preview copy", "env_preview", 0),
      firewallRow("pol_2", "Prod copy 2", "env_prod", 1),
    ];

    const environments = environmentsInMutations(rows);

    expect(environments).toHaveLength(2);
    expect(environments).toContainEqual({
      environmentId: "env_prod",
      projectId: "proj_KEBAP",
      appId: "app_KEBAP",
    });
    expect(environments).toContainEqual({
      environmentId: "env_preview",
      projectId: "proj_KEBAP",
      appId: "app_KEBAP",
    });
  });
});

describe("dispatchUpdate", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends policyId as a sibling field, not part of the policy body", async () => {
    const requests: { url: string; body: unknown }[] = [];
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

    await dispatchUpdate(firewallRow("pol_1", "KEBAP", "env_1", 0));

    expect(requests).toHaveLength(1);
    expect(requests[0].url).toBe("http://localhost:3000/proxy/v2/gateway.updatePolicy");
    expect(requests[0].body).toMatchObject({
      project: "proj_KEBAP",
      app: "app_KEBAP",
      environment: "env_1",
      policyId: "pol_1",
      name: "KEBAP",
      enabled: true,
      firewall: { action: "ACTION_DENY" },
    });
    expect(requests[0].body).not.toHaveProperty("id");
  });

  it("sends an empty match array rather than omitting it, so save clears prior conditions", async () => {
    const requests: { body: unknown }[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init);
        requests.push({ body: await request.clone().json() });
        return new Response(JSON.stringify({ meta: { requestId: "req_1" }, data: {} }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );

    const row = { ...firewallRow("pol_1", "KEBAP", "env_1", 0), match: [] };
    await dispatchUpdate(row);

    expect(requests[0].body).toHaveProperty("match", []);
  });
});

// A reorder sends the whole list to setPolicies, which is a full replace. The
// list must hold every policy of the environment, also when two policies share
// a name.
describe("reorderPolicies", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  function captureRequests() {
    const bodies: { policies: { id?: string; name: string }[] }[] = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const request = input instanceof Request ? input : new Request(input, init);
        bodies.push(await request.clone().json());
        return new Response(JSON.stringify({ meta: { requestId: "req_1" }, data: {} }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );
    return bodies;
  }

  it("sends every policy, in the given order, when two share a name", async () => {
    const bodies = captureRequests();
    const ordered = [
      firewallRow("pol_3", "Auth", "env_1", 0),
      firewallRow("pol_1", "Ratelimit", "env_1", 1),
      firewallRow("pol_2", "Ratelimit", "env_1", 2),
    ];

    await reorderPolicies([
      { environmentId: "env_1", projectId: "proj_KEBAP", appId: "app_KEBAP", policies: ordered },
    ]);

    expect(bodies).toHaveLength(1);
    expect(bodies[0].policies.map((p) => p.name)).toEqual(["Auth", "Ratelimit", "Ratelimit"]);
  });

  it("sends one request per environment", async () => {
    const bodies = captureRequests();

    await reorderPolicies([
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
    ]);

    expect(bodies).toHaveLength(2);
  });
});
