import {
  type PolicyListReplacement,
  type PolicyLists,
  type PolicyRow,
  type PolicyScope,
  createPolicyWriter,
} from "@/lib/collections/deploy/policies";
import { type PolicyInput, policyMatchKey } from "@/lib/collections/deploy/policies.schema";
import { describe, expect, it } from "vitest";
import { type Env, type MergedPolicy, mergePolicies } from "../components/list/merge";
import {
  insertPolicy,
  removePolicy,
  reorderPolicies,
  setEnabled,
  updatePolicy,
} from "./policy-edits";
import {
  NO_PENDING_SWITCHES,
  type PendingSwitchEvent,
  policySwitches,
  reducePendingSwitches,
} from "./policy-switches";

const LABELS = { loading: "Saving...", success: "Saved", error: "Failed" };
const SCOPE: PolicyScope = {
  projectId: "proj_KEBAP",
  appId: "app_KEBAP",
  environments: { production: "env_prod", preview: "env_prev" },
};

function firewall(name: string, enabled = true): PolicyInput {
  return { name, enabled, type: "firewall", firewall: { action: "ACTION_DENY" } };
}

const key = (name: string) => policyMatchKey("firewall", name);

/**
 * Stands in for the gateway: `setPolicies` stores the list and assigns every
 * policy a new id, and the next read returns those ids.
 */
function fakeGateway(initial: Record<"production" | "preview", PolicyInput[]>) {
  let nextId = 0;
  const stored: PolicyLists = { production: [], preview: [] };
  const writes: PolicyListReplacement[][] = [];
  const notices: string[] = [];
  const pending: { answer: () => void; fail: () => void }[] = [];

  const store = (environmentId: string, policies: PolicyInput[]) => {
    const env = environmentId === "env_prod" ? "production" : "preview";
    stored[env] = policies.map(
      (p, index): PolicyRow => ({
        ...p,
        id: `pol_${++nextId}`,
        environmentId,
        projectId: SCOPE.projectId,
        appId: SCOPE.appId,
        _order: index,
      }),
    );
  };
  store("env_prod", initial.production);
  store("env_prev", initial.preview);

  const write = createPolicyWriter({
    read: () => ({
      type: "lists",
      lists: { production: [...stored.production], preview: [...stored.preview] },
    }),
    write: (replacements) =>
      new Promise<void>((resolve, reject) => {
        pending.push({
          answer: () => {
            writes.push(replacements);
            for (const r of replacements) {
              store(r.environmentId, r.policies);
            }
            resolve();
          },
          fail: () => reject(new Error("gateway down")),
        });
      }),
    notify: (message) => notices.push(message),
  });

  return {
    write,
    writes,
    notices,
    stored,
    serverAnswers: () => settleNext(() => pending.shift()?.answer()),
    serverFails: () => settleNext(() => pending.shift()?.fail()),
  };
}

async function settleNext(settle: () => void) {
  await Promise.resolve();
  settle();
  for (let i = 0; i < 4; i++) {
    await Promise.resolve();
  }
}

const names = (rows: { name: string; enabled: boolean }[]) =>
  rows.map((r) => `${r.name}${r.enabled ? "" : " (off)"}`);

describe("policy writer", () => {
  it("runs edits in order, each against the lists the previous write left behind", async () => {
    const gw = fakeGateway({ production: [firewall("A"), firewall("B")], preview: [] });

    const first = gw.write(SCOPE, removePolicy(key("A")), LABELS);
    const second = gw.write(SCOPE, insertPolicy(firewall("C"), ["production"]), LABELS);
    await gw.serverAnswers();
    await gw.serverAnswers();

    expect(await first).toBe(true);
    expect(await second).toBe(true);
    expect(gw.writes.map((w) => w[0].policies.map((p) => p.name))).toEqual([["B"], ["B", "C"]]);
    expect(names(gw.stored.production)).toEqual(["B", "C"]);
  });

  it("a toggle and an edit queued during a reorder both land on the regenerated ids", async () => {
    const gw = fakeGateway({
      production: [firewall("Auth"), firewall("Limit")],
      preview: [firewall("Auth")],
    });
    const before = gw.stored.production.map((r) => r.id);

    const reorder = gw.write(SCOPE, reorderPolicies(key("Limit"), key("Auth")), LABELS);
    const toggle = gw.write(SCOPE, setEnabled(key("Auth"), "production", false), LABELS);
    const edit = gw.write(
      SCOPE,
      updatePolicy(key("Auth"), {
        ...firewall("Auth v2"),
        match: [{ path: { path: { prefix: "/v2" } } }],
      }),
      LABELS,
    );
    await gw.serverAnswers();
    await gw.serverAnswers();
    await gw.serverAnswers();

    expect([await reorder, await toggle, await edit]).toEqual([true, true, true]);
    expect(gw.notices).toEqual([]);
    expect(gw.stored.production.map((r) => r.id)).not.toEqual(before);
    expect(names(gw.stored.production)).toEqual(["Limit", "Auth v2 (off)"]);
    expect(gw.stored.production[1].match).toEqual([{ path: { path: { prefix: "/v2" } } }]);
    expect(names(gw.stored.preview)).toEqual(["Auth v2"]);
  });

  it("rejects an edit to a policy that an earlier queued delete removed", async () => {
    const gw = fakeGateway({ production: [firewall("A")], preview: [] });

    const remove = gw.write(SCOPE, removePolicy(key("A")), LABELS);
    const toggle = gw.write(SCOPE, setEnabled(key("A"), "production", false), LABELS);
    await gw.serverAnswers();
    await gw.serverAnswers();

    expect(await remove).toBe(true);
    expect(await toggle).toBe(false);
    expect(gw.notices).toEqual(["This policy no longer exists."]);
    expect(gw.writes).toHaveLength(1);
  });

  it("turning a policy on where it is absent copies it there, and the two merge into one", async () => {
    const gw = fakeGateway({ production: [firewall("A")], preview: [firewall("X")] });

    const copy = gw.write(SCOPE, setEnabled(key("A"), "preview", true), LABELS);
    await gw.serverAnswers();

    expect(await copy).toBe(true);
    expect(names(gw.stored.preview)).toEqual(["X", "A"]);
    expect(names(gw.stored.production)).toEqual(["A"]);
    const merged = mergePolicies(gw.stored.production, gw.stored.preview);
    expect(merged.map((m) => [m.name, m.production?.enabled, m.preview?.enabled])).toEqual([
      ["A", true, true],
      ["X", undefined, true],
    ]);
  });

  it("turning a policy off keeps its copy, so turning it on again keeps its place", async () => {
    const gw = fakeGateway({ production: [], preview: [firewall("A"), firewall("B")] });

    const off = gw.write(SCOPE, setEnabled(key("A"), "preview", false), LABELS);
    const on = gw.write(SCOPE, setEnabled(key("A"), "preview", true), LABELS);
    await gw.serverAnswers();
    expect(names(gw.stored.preview)).toEqual(["A (off)", "B"]);
    await gw.serverAnswers();

    expect([await off, await on]).toEqual([true, true]);
    expect(names(gw.stored.preview)).toEqual(["A", "B"]);
  });

  it("turning a policy off where it is absent writes nothing", async () => {
    const gw = fakeGateway({ production: [firewall("A")], preview: [] });

    expect(await gw.write(SCOPE, setEnabled(key("A"), "preview", false), LABELS)).toBe(true);
    expect(gw.writes).toEqual([]);
  });

  it("refuses to copy a policy whose name repeats, since the copy would not pair", async () => {
    const gw = fakeGateway({
      production: [firewall("Auth"), firewall("Auth", false)],
      preview: [firewall("Auth")],
    });
    const second = mergePolicies(gw.stored.production, gw.stored.preview)[1];

    const copy = gw.write(SCOPE, setEnabled(second.key, "preview", true), LABELS);

    expect(await copy).toBe(false);
    expect(gw.notices).toEqual([
      "Another policy of this type has the same name. Rename one of them first.",
    ]);
    expect(gw.writes).toHaveLength(0);
  });

  it("rejects an insert past the environment's capacity without writing", async () => {
    const gw = fakeGateway({
      production: Array.from({ length: 50 }, (_, i) => firewall(`P${i}`)),
      preview: [],
    });

    const insert = gw.write(
      SCOPE,
      insertPolicy(firewall("One more"), ["production", "preview"]),
      LABELS,
    );

    expect(await insert).toBe(false);
    expect(gw.notices).toEqual(["An environment holds at most 50 policies."]);
    expect(gw.writes).toHaveLength(0);
  });

  it("reports a failed write and keeps serving the edits behind it", async () => {
    const gw = fakeGateway({ production: [firewall("A")], preview: [] });
    let failures = 0;
    const flaky = createPolicyWriter({
      read: () => ({ type: "lists", lists: gw.stored }),
      write: () => Promise.reject(new Error(`boom ${++failures}`)),
      notify: () => undefined,
    });

    const failed = flaky(SCOPE, removePolicy(key("A")), LABELS);
    const next = flaky(SCOPE, setEnabled(key("A"), "production", false), LABELS);

    expect(await failed).toBe(false);
    expect(await next).toBe(false);
    expect(failures).toBe(2);
  });
});

function switchesPage(gw: ReturnType<typeof fakeGateway>) {
  let pending = NO_PENDING_SWITCHES;
  let clicks = 0;
  const dispatch = (event: PendingSwitchEvent) => {
    pending = reducePendingSwitches(pending, event);
  };
  const find = (match: (m: MergedPolicy) => boolean) => {
    const policy = mergePolicies(gw.stored.production, gw.stored.preview).find(match);
    if (!policy) {
      throw new Error("no such policy");
    }
    return policy;
  };
  const clickKey = (policyKey: string, env: Env) => {
    const policy = find((m) => m.key === policyKey);
    const on = !policySwitches(policy, pending)[env];
    const click = ++clicks;
    dispatch({ type: "click", key: policyKey, env, on, click });
    return gw
      .write(SCOPE, setEnabled(policyKey, env, on), LABELS)
      .then((ok) => dispatch({ type: "settle", key: policyKey, env, click, ok }));
  };
  return {
    shown: (name: string) =>
      policySwitches(
        find((m) => m.name === name),
        pending,
      ),
    shownKey: (policyKey: string) =>
      policySwitches(
        find((m) => m.key === policyKey),
        pending,
      ),
    click: (name: string, env: Env) => clickKey(find((m) => m.name === name).key, env),
    clickKey,
    listsSync: () => dispatch({ type: "sync" }),
  };
}

describe("policy switches", () => {
  it("flips at once and keeps the flip once the write lands", async () => {
    const gw = fakeGateway({ production: [firewall("A")], preview: [firewall("A")] });
    const page = switchesPage(gw);

    const click = page.click("A", "production");
    expect(page.shown("A")).toEqual({ production: false, preview: true });
    expect(gw.writes).toHaveLength(0);

    await gw.serverAnswers();
    await click;
    expect(page.shown("A")).toEqual({ production: false, preview: true });

    page.listsSync();
    expect(page.shown("A")).toEqual({ production: false, preview: true });
    expect(names(gw.stored.production)).toEqual(["A (off)"]);
  });

  it("rolls back to the server state when the write fails", async () => {
    const gw = fakeGateway({ production: [firewall("A")], preview: [] });
    const page = switchesPage(gw);

    const click = page.click("A", "production");
    expect(page.shown("A")).toEqual({ production: false, preview: false });

    await gw.serverFails();
    await click;
    expect(page.shown("A")).toEqual({ production: true, preview: false });
    expect(names(gw.stored.production)).toEqual(["A"]);
  });

  it("two fast clicks on one switch end where the last click left it", async () => {
    const gw = fakeGateway({ production: [firewall("A")], preview: [] });
    const page = switchesPage(gw);

    const first = page.click("A", "production");
    const second = page.click("A", "production");
    expect(page.shown("A")).toEqual({ production: true, preview: false });

    await gw.serverAnswers();
    await first;
    expect(page.shown("A")).toEqual({ production: true, preview: false });
    await gw.serverAnswers();
    await second;
    page.listsSync();

    expect(page.shown("A")).toEqual({ production: true, preview: false });
    expect(gw.writes.map((w) => names(w[0].policies))).toEqual([["A (off)"], ["A"]]);
  });

  it("two fast clicks on an unnamed policy both land", async () => {
    const gw = fakeGateway({ production: [firewall("A"), firewall("")], preview: [] });
    const page = switchesPage(gw);
    const unnamed = mergePolicies(gw.stored.production, gw.stored.preview)[1].key;

    const first = page.clickKey(unnamed, "production");
    const second = page.clickKey(unnamed, "production");
    await gw.serverAnswers();
    await first;
    await gw.serverAnswers();
    await second;
    page.listsSync();

    expect(gw.notices).toEqual([]);
    expect(gw.writes.map((w) => names(w[0].policies))).toEqual([
      ["A", " (off)"],
      ["A", ""],
    ]);
    expect(page.shownKey(unnamed)).toEqual({ production: true, preview: false });
  });

  it("a failed first click does not undo a second click that landed", async () => {
    const gw = fakeGateway({ production: [firewall("A")], preview: [] });
    const page = switchesPage(gw);

    const first = page.click("A", "production");
    const second = page.click("A", "production");

    await gw.serverFails();
    await first;
    expect(page.shown("A")).toEqual({ production: true, preview: false });
    await gw.serverAnswers();
    await second;
    page.listsSync();

    expect(page.shown("A")).toEqual({ production: true, preview: false });
    expect(names(gw.stored.production)).toEqual(["A"]);
  });

  it("fast clicks on two policies each land", async () => {
    const gw = fakeGateway({
      production: [firewall("A"), firewall("B")],
      preview: [firewall("B")],
    });
    const page = switchesPage(gw);

    const a = page.click("A", "production");
    const b = page.click("B", "preview");
    expect([page.shown("A"), page.shown("B")]).toEqual([
      { production: false, preview: false },
      { production: true, preview: false },
    ]);

    await gw.serverAnswers();
    await gw.serverAnswers();
    await Promise.all([a, b]);
    page.listsSync();

    expect([page.shown("A"), page.shown("B")]).toEqual([
      { production: false, preview: false },
      { production: true, preview: false },
    ]);
    expect(names(gw.stored.production)).toEqual(["A (off)", "B"]);
    expect(names(gw.stored.preview)).toEqual(["B (off)"]);
  });

  it("turning a switch on where the policy is absent shows on at once and lands as one policy", async () => {
    const gw = fakeGateway({ production: [firewall("A")], preview: [] });
    const page = switchesPage(gw);

    const click = page.click("A", "preview");
    expect(page.shown("A")).toEqual({ production: true, preview: true });
    expect(gw.writes).toHaveLength(0);

    await gw.serverAnswers();
    await click;
    page.listsSync();

    expect(page.shown("A")).toEqual({ production: true, preview: true });
    expect(mergePolicies(gw.stored.production, gw.stored.preview)).toHaveLength(1);
  });

  it("rolls back a switch the edit rejects", async () => {
    const gw = fakeGateway({
      production: [firewall("Auth"), firewall("Auth", false)],
      preview: [],
    });
    const page = switchesPage(gw);
    const second = mergePolicies(gw.stored.production, gw.stored.preview)[1];

    const click = page.clickKey(second.key, "preview");
    expect(page.shownKey(second.key)).toEqual({ production: false, preview: true });

    await click;
    expect(page.shownKey(second.key)).toEqual({ production: false, preview: false });
    expect(gw.writes).toEqual([]);
  });
});
