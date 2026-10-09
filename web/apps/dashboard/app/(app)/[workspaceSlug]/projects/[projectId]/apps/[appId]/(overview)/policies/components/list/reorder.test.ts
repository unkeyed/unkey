import type { PolicyRow } from "@/lib/collections/deploy/policies";
import { describe, expect, it } from "vitest";
import { type Env, mergePolicies } from "./merge";
import { movePolicy } from "./reorder";

function firewall(id: string, name: string): PolicyRow {
  return {
    id,
    name,
    enabled: true,
    type: "firewall",
    firewall: { action: "ACTION_DENY" },
    environmentId: "env_1",
    projectId: "proj_KEBAP",
    appId: "app_KEBAP",
  };
}

function move(rowsByEnv: Record<Env, PolicyRow[]>, from: string, to: string) {
  const merged = mergePolicies(rowsByEnv.production, rowsByEnv.preview);
  const index = (name: string) => merged.findIndex((m) => m.name === name);
  return Object.fromEntries(
    movePolicy(merged, rowsByEnv, index(from), index(to)).map(({ env, rows }) => [
      env,
      rows.map((r) => r.id),
    ]),
  );
}

describe("movePolicy", () => {
  it("moves a shared row up in each environment and keeps a preview-only row in place", () => {
    const lists = move(
      {
        production: [firewall("prod_a", "A"), firewall("prod_b", "B")],
        preview: [firewall("prev_x", "X"), firewall("prev_a", "A"), firewall("prev_b", "B")],
      },
      "B",
      "A",
    );

    expect(lists).toEqual({
      production: ["prod_b", "prod_a"],
      preview: ["prev_x", "prev_b", "prev_a"],
    });
  });

  it("moves a shared row down in each environment and keeps a preview-only row in place", () => {
    const lists = move(
      {
        production: [firewall("prod_a", "A"), firewall("prod_b", "B"), firewall("prod_c", "C")],
        preview: [firewall("prev_a", "A"), firewall("prev_x", "X"), firewall("prev_b", "B")],
      },
      "A",
      "B",
    );

    expect(lists).toEqual({
      production: ["prod_b", "prod_a", "prod_c"],
      preview: ["prev_x", "prev_b", "prev_a"],
    });
  });

  it("leaves preview alone when the row is already above every row it passed", () => {
    const lists = move(
      {
        production: [firewall("prod_a", "A"), firewall("prod_b", "B"), firewall("prod_c", "C")],
        preview: [firewall("prev_c", "C"), firewall("prev_b", "B"), firewall("prev_a", "A")],
      },
      "C",
      "A",
    );

    expect(lists).toEqual({ production: ["prod_c", "prod_a", "prod_b"] });
  });

  it("leaves an environment alone when the row passed none of its rows", () => {
    const lists = move(
      {
        production: [firewall("prod_a", "A"), firewall("prod_p", "P"), firewall("prod_b", "B")],
        preview: [firewall("prev_b", "B"), firewall("prev_a", "A")],
      },
      "B",
      "P",
    );

    expect(lists).toEqual({ production: ["prod_a", "prod_b", "prod_p"] });
  });

  it("does not rewrite an environment the moved row is not in", () => {
    const lists = move(
      {
        production: [firewall("prod_a", "A"), firewall("prod_b", "B")],
        preview: [firewall("prev_x", "X")],
      },
      "B",
      "A",
    );

    expect(lists).toEqual({ production: ["prod_b", "prod_a"] });
  });

  it("moves a preview-only row relative to its new preview neighbour", () => {
    const lists = move(
      {
        production: [firewall("prod_a", "A"), firewall("prod_b", "B")],
        preview: [firewall("prev_b", "B"), firewall("prev_x", "X"), firewall("prev_a", "A")],
      },
      "X",
      "B",
    );

    expect(lists).toEqual({ preview: ["prev_x", "prev_b", "prev_a"] });
  });
});
