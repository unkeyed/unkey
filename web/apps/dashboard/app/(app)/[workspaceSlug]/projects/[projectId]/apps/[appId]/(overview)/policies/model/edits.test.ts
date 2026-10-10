import type { PolicyRow } from "@/lib/collections/deploy/policies";
import { POLICY_ERRORS, type PolicyInput } from "@/lib/collections/deploy/policies.schema";
import { describe, expect, it } from "vitest";
import {
  type PolicyEdit,
  type PolicyListsScope,
  type PolicyOp,
  insertPolicy,
  removePolicy,
  reorderPolicies,
  setEnabled,
  updatePolicy,
} from "./edits";
import { type Env, mergePolicies } from "./merge";

const SCOPE: PolicyListsScope = {
  projectId: "proj_KEBAP",
  appId: "app_KEBAP",
  environments: { production: "env_prod", preview: "env_prev" },
};

function firewall(name: string, enabled = true): PolicyInput {
  return { name, enabled, type: "firewall", firewall: { action: "ACTION_DENY" } };
}

function lists(production: PolicyInput[], preview: PolicyInput[]): Record<Env, PolicyRow[]> {
  const rows = (env: Env, policies: PolicyInput[]) =>
    policies.map(
      (p, index): PolicyRow => ({
        ...p,
        id: `pol_${env}_${index}`,
        environmentId: SCOPE.environments[env],
        projectId: SCOPE.projectId,
        appId: SCOPE.appId,
        _order: index,
      }),
    );
  return { production: rows("production", production), preview: rows("preview", preview) };
}

const keyOf = (state: Record<Env, PolicyRow[]>, name: string) => {
  const row = mergePolicies(state.production, state.preview).find((m) => m.name === name);
  if (!row) {
    throw new Error(`no policy named ${name}`);
  }
  return row.key;
};

function ops(edit: PolicyEdit, state: Record<Env, PolicyRow[]>): PolicyOp[] {
  const result = edit(state, SCOPE);
  if (result.type === "reject") {
    throw new Error(result.message);
  }
  return result.ops;
}

const rejection = (edit: PolicyEdit, state: Record<Env, PolicyRow[]>) => {
  const result = edit(state, SCOPE);
  return result.type === "reject" ? result.message : null;
};

describe("insertPolicy", () => {
  it("adds a row at the end of each target environment", () => {
    const state = lists([firewall("A")], []);

    expect(ops(insertPolicy(firewall("B"), ["production", "preview"]), state)).toEqual([
      {
        type: "insert",
        row: expect.objectContaining({ name: "B", environmentId: "env_prod", _order: 1 }),
      },
      {
        type: "insert",
        row: expect.objectContaining({ name: "B", environmentId: "env_prev", _order: 0 }),
      },
    ]);
  });

  it("rejects an insert past the environment's capacity", () => {
    const state = lists(
      Array.from({ length: 50 }, (_, i) => firewall(`P${i}`)),
      [],
    );

    expect(rejection(insertPolicy(firewall("One more"), ["production"]), state)).toBe(
      POLICY_ERRORS.atCapacity,
    );
  });

  it("rejects a second policy of the same type and name, also one still being added", () => {
    const state = lists([], []);
    const [first] = ops(insertPolicy(firewall("Auth"), ["preview"]), state);
    if (first.type !== "insert") {
      throw new Error("expected an insert");
    }
    state.preview.push(first.row);

    expect(rejection(insertPolicy(firewall("Auth "), ["production"]), state)).toMatch(
      /already exists/,
    );
  });
});

describe("setEnabled", () => {
  it("updates the copy in place when the environment has one", () => {
    const state = lists([firewall("A")], [firewall("A")]);

    expect(ops(setEnabled(keyOf(state, "A"), "preview", false), state)).toEqual([
      { type: "update", row: state.preview[0], patch: { enabled: false } },
    ]);
  });

  it("writes nothing when the copy already has that state", () => {
    const state = lists([firewall("A")], []);

    expect(ops(setEnabled(keyOf(state, "A"), "production", true), state)).toEqual([]);
    expect(ops(setEnabled(keyOf(state, "A"), "preview", false), state)).toEqual([]);
  });

  it("copies the policy into an environment where it is absent", () => {
    const state = lists([{ ...firewall("A"), match: [{ path: { path: { prefix: "/v1" } } }] }], []);

    const [copy] = ops(setEnabled(keyOf(state, "A"), "preview", true), state);

    expect(copy).toEqual({
      type: "insert",
      row: expect.objectContaining({
        name: "A",
        enabled: true,
        environmentId: "env_prev",
        match: [{ path: { path: { prefix: "/v1" } } }],
      }),
    });
  });

  it("refuses to copy a policy whose name repeats, since the copy would not pair", () => {
    const state = lists([firewall("Auth"), firewall("Auth", false)], []);
    const second = mergePolicies(state.production, state.preview)[1].key;

    expect(rejection(setEnabled(second, "preview", true), state)).toBe(POLICY_ERRORS.sharedName);
  });

  it("rejects a toggle on a policy that is gone", () => {
    expect(rejection(setEnabled("firewall:A", "production", false), lists([], []))).toBe(
      POLICY_ERRORS.gone,
    );
  });
});

describe("updatePolicy", () => {
  it("writes the new values to every copy and keeps each copy's on or off state", () => {
    const state = lists([firewall("Auth")], [firewall("Auth", false)]);
    const next = { ...firewall("Auth v2"), match: [{ path: { path: { prefix: "/v2" } } }] };

    expect(ops(updatePolicy(keyOf(state, "Auth"), next), state)).toEqual([
      { type: "update", row: state.production[0], patch: { ...next, enabled: true } },
      { type: "update", row: state.preview[0], patch: { ...next, enabled: false } },
    ]);
  });

  it("rejects a rename onto another policy of the same type", () => {
    const state = lists([firewall("Auth"), firewall("Limit")], []);

    expect(rejection(updatePolicy(keyOf(state, "Auth"), firewall("Limit")), state)).toMatch(
      /already exists/,
    );
  });
});

describe("removePolicy", () => {
  it("deletes the copy in every environment", () => {
    const state = lists([firewall("A")], [firewall("A")]);

    expect(ops(removePolicy(keyOf(state, "A")), state)).toEqual([
      { type: "delete", row: state.production[0] },
      { type: "delete", row: state.preview[0] },
    ]);
  });
});

describe("reorderPolicies", () => {
  it("gives the moved row an order between its new neighbours in each environment", () => {
    const state = lists(
      [firewall("A"), firewall("B"), firewall("C")],
      [firewall("X"), firewall("B"), firewall("C")],
    );

    expect(ops(reorderPolicies(keyOf(state, "C"), keyOf(state, "B")), state)).toEqual([
      { type: "update", row: state.production[2], patch: { _order: 0.5 } },
      { type: "update", row: state.preview[2], patch: { _order: 0.5 } },
    ]);
  });

  it("moves a row to the top below zero", () => {
    const state = lists([firewall("A"), firewall("B")], []);

    expect(ops(reorderPolicies(keyOf(state, "B"), keyOf(state, "A")), state)).toEqual([
      { type: "update", row: state.production[1], patch: { _order: -1 } },
    ]);
  });
});
