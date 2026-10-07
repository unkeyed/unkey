import type { PolicyRow } from "@/lib/collections/deploy/policies";
import { describe, expect, it } from "vitest";
import { policyRowState } from "./row-state";

const envs = {
  production: { id: "env_prod", slug: "production" },
  preview: { id: "env_prev", slug: "preview" },
};

function firewall(id: string, enabled: boolean): PolicyRow {
  return {
    id,
    name: "Block admin",
    enabled,
    type: "firewall",
    firewall: { action: "ACTION_DENY" },
    environmentId: "env_1",
    projectId: "proj_1",
    appId: "app_1",
  };
}

describe("policyRowState", () => {
  it("on in production and off in preview: not dimmed", () => {
    const production = firewall("pol_prod", true);
    const preview = firewall("pol_prev", false);
    expect(
      policyRowState(
        { key: "k", name: "Block admin", type: "firewall", production, preview },
        envs,
      ),
    ).toEqual({
      dimmed: false,
      name: { type: "named", text: "Block admin" },
      environments: [
        { type: "on", env: "production", slug: "production" },
        { type: "off", env: "preview", slug: "preview" },
      ],
    });
  });

  it("off everywhere it exists: dimmed", () => {
    const preview = firewall("pol_prev", false);
    expect(
      policyRowState(
        { key: "k", name: "Block admin", type: "firewall", production: null, preview },
        envs,
      ),
    ).toEqual({
      dimmed: true,
      name: { type: "named", text: "Block admin" },
      environments: [{ type: "off", env: "preview", slug: "preview" }],
    });
  });

  it("untitled and in no environment", () => {
    expect(
      policyRowState(
        { key: "k", name: "", type: "firewall", production: null, preview: null },
        envs,
      ),
    ).toEqual({ dimmed: true, name: { type: "untitled" }, environments: [] });
  });
});
