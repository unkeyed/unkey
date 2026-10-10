import { describe, expect, it, vi } from "vitest";
import { type Policy, type PolicyInput, fromWirePolicy, policyMatchKey } from "./policies.schema";

type Stored = Policy & { id: string };

const server: Record<string, Stored[]> = {};
const failing = { on: false };
let nextId = 0;

function store(environment: string, policies: PolicyInput[]) {
  server[environment] = policies.map((p) => ({ ...p, id: `pol_${++nextId}` }));
}

function wire({ type: _type, ...policy }: Stored) {
  return policy;
}

const settle = () => new Promise((resolve) => setTimeout(resolve, 5));

const gateway = {
  listPolicies: vi.fn(async ({ environment }: { environment: string }) => ({
    data: server[environment].map(wire),
  })),
  setPolicies: vi.fn(async (body: { environment: string; policies: PolicyInput[] }) => {
    await settle();
    if (failing.on) {
      throw new Error("gateway down");
    }
    store(body.environment, body.policies);
    return {};
  }),
  updatePolicy: vi.fn(
    async ({
      project: _project,
      app: _app,
      environment,
      policyId,
      ...body
    }: {
      project: string;
      app: string;
      environment: string;
      policyId: string;
    }) => {
      await settle();
      if (failing.on) {
        throw new Error("gateway down");
      }
      const row = server[environment].find((p) => p.id === policyId);
      if (!row) {
        throw new Error("not found");
      }
      Object.assign(row, body);
      return {};
    },
  ),
};

vi.mock("@/lib/unkey-client", () => ({
  getUnkeyClient: () => ({ gateway }),
  getErrorToast: () => ({ message: "", description: "" }),
}));
vi.mock("@unkey/ui", () => ({ toast: { promise: vi.fn(), error: vi.fn() } }));
vi.mock("../client", async () => {
  const { QueryClient } = await import("@tanstack/query-core");
  return { queryClient: new QueryClient() };
});

const { and, createLiveQueryCollection, createPacedMutations, eq, queueStrategy } = await import(
  "@tanstack/react-db"
);
const { nextPolicyList, persistPolicyMutations, policies, resolveRow, rowKey } = await import(
  "./policies"
);
type PolicyRow = import("./policies").PolicyRow;

const LABELS = { loading: "Saving...", success: "Saved", error: "Failed" };

function firewall(name: string, enabled = true): PolicyInput {
  return { name, enabled, type: "firewall", firewall: { action: "ACTION_DENY" } };
}

function row(id: string, name: string, order: number): PolicyRow {
  return {
    ...firewall(name),
    id,
    environmentId: "env_prod",
    projectId: "proj_KEBAP",
    appId: "app_KEBAP",
    _order: order,
  };
}

describe("rowKey", () => {
  it("scopes a policy id to its environment, so two copies do not collide", () => {
    expect(rowKey("env_prod", "pol_1")).not.toBe(rowKey("env_preview", "pol_1"));
  });
});

describe("resolveRow", () => {
  it("finds a row by id, then by a unique type and name once the ids changed", () => {
    const fresh = [row("pol_9", "A", 0), row("pol_8", "B", 1)];

    expect(resolveRow(fresh, row("pol_8", "renamed", 0))?.name).toBe("B");
    expect(resolveRow(fresh, row("pol_1", "A", 1))?.id).toBe("pol_9");
    expect(resolveRow([...fresh, row("pol_7", "A", 2)], row("pol_1", "A", 0))).toBeNull();
  });
});

describe("nextPolicyList", () => {
  const fresh = [row("pol_4", "A", 0), row("pol_5", "B", 1), row("pol_6", "C", 2)];

  it("moves a row by its new order and keeps the fresh ids", () => {
    const moved = { ...row("pol_1", "C", 2), _order: 0.5 };
    const next = nextPolicyList(fresh, [
      { type: "update", original: row("pol_1", "C", 2), modified: moved, changes: { _order: 0.5 } },
    ]);

    expect(next.map((r) => `${r.name}:${r.id}`)).toEqual(["A:pol_4", "C:pol_6", "B:pol_5"]);
  });

  it("appends an insert and deletes by the fresh row", () => {
    const next = nextPolicyList(fresh, [
      {
        type: "delete",
        original: row("pol_2", "B", 1),
        modified: row("pol_2", "B", 1),
        changes: {},
      },
      { type: "insert", original: {}, modified: row("new:firewall:D", "D", 3), changes: {} },
    ]);

    expect(next.map((r) => r.name)).toEqual(["A", "C", "D"]);
  });

  it("refuses an insert whose name another write added meanwhile", () => {
    expect(() =>
      nextPolicyList(fresh, [
        { type: "insert", original: {}, modified: row("new:firewall:B", "B", 3), changes: {} },
      ]),
    ).toThrow("same name");
  });
});

async function page(environment: string, production: PolicyInput[]) {
  store(environment, production);
  const list = createLiveQueryCollection((q) =>
    q
      .from({ p: policies })
      .where(({ p }) =>
        and(
          eq(p.projectId, "proj_KEBAP"),
          eq(p.appId, "app_KEBAP"),
          eq(p.environmentId, environment),
        ),
      )
      .orderBy(({ p }) => p._order),
  );
  await list.preload();
  await vi.waitFor(() => expect(list.toArray).toHaveLength(production.length));
  const strategy = queueStrategy();
  const write = (onMutate: () => void) =>
    createPacedMutations<null, PolicyRow>({
      strategy,
      onMutate,
      mutationFn: ({ transaction }) => persistPolicyMutations(transaction.mutations, LABELS),
    })(null);
  const shown = () => list.toArray.map((r) => `${r.name}${r.enabled ? "" : " (off)"}`);
  const keyOf = (name: string) => {
    const found = list.toArray.find((r) => r.name === name);
    if (!found) {
      throw new Error(`no row ${name}`);
    }
    return rowKey(found.environmentId, found.id);
  };
  return { write, shown, keyOf };
}

describe("queued policy writes", () => {
  it("shows a toggle at once, keeps the policy id, and rolls the toggle back when the write fails", async () => {
    const { write, shown, keyOf } = await page("env_toggle", [firewall("A")]);
    const id = server.env_toggle[0].id;

    const off = write(() =>
      policies.update(keyOf("A"), (d) => {
        d.enabled = false;
      }),
    );
    expect(shown()).toEqual(["A (off)"]);
    await off.isPersisted.promise;
    expect(server.env_toggle.map((p) => [p.id, p.enabled])).toEqual([[id, false]]);

    failing.on = true;
    const on = write(() =>
      policies.update(keyOf("A"), (d) => {
        d.enabled = true;
      }),
    );
    expect(shown()).toEqual(["A"]);
    await expect(on.isPersisted.promise).rejects.toThrow("gateway down");
    failing.on = false;
    expect(shown()).toEqual(["A (off)"]);
  });

  it("lands a toggle queued behind a reorder on the ids the reorder regenerated", async () => {
    const { write, shown, keyOf } = await page("env_reorder", [
      firewall("Auth"),
      firewall("Limit"),
    ]);

    const reorder = write(() =>
      policies.update(keyOf("Limit"), (d) => {
        d._order = -1;
      }),
    );
    const toggle = write(() =>
      policies.update(keyOf("Auth"), (d) => {
        d.enabled = false;
      }),
    );
    expect(shown()).toEqual(["Limit", "Auth (off)"]);

    await reorder.isPersisted.promise;
    await toggle.isPersisted.promise;

    expect(server.env_reorder.map((p) => `${p.name}${p.enabled ? "" : " (off)"}`)).toEqual([
      "Limit",
      "Auth (off)",
    ]);
    await vi.waitFor(() => expect(shown()).toEqual(["Limit", "Auth (off)"]));
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
