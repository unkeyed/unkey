import { describe, expect, it } from "vitest";
import { checkoutReturnPath, parseReturnPath, parseUpgradeResult } from "./upgrade-result";

function params(path: string) {
  return new URL(path, "https://app.unkey.com").searchParams;
}

describe("parseReturnPath", () => {
  it.each([
    ["https://evil.com"],
    ["//evil.com"],
    ["/other-ws/settings"],
    ["/ws/../x"],
    ["/ws/a?b=c"],
    ["/ws//settings"],
    ["/wsx/settings"],
    [""],
    [null],
    [undefined],
  ])("rejects %s", (raw) => {
    expect(parseReturnPath(raw, "ws")).toBe(null);
  });

  it.each([["/ws"], ["/ws/settings/team"], ["/ws/projects/proj_123"]])("accepts %s", (raw) => {
    expect(parseReturnPath(raw, "ws")).toBe(raw);
  });
});

describe("parseUpgradeResult", () => {
  it("reads every result kind", () => {
    expect(parseUpgradeResult(params("/ws?upgraded=compute&plan=business"))).toEqual({
      kind: "compute",
      plan: "business",
    });
    expect(parseUpgradeResult(params("/ws?upgraded=api"))).toEqual({ kind: "api" });
  });

  it("rejects compute with an unknown or missing plan", () => {
    expect(parseUpgradeResult(params("/ws?upgraded=compute&plan=enterprise"))).toBe(null);
    expect(parseUpgradeResult(params("/ws?upgraded=compute"))).toBe(null);
  });

  it("rejects unknown or missing results", () => {
    expect(parseUpgradeResult(params("/ws?upgraded=gold"))).toBe(null);
    expect(parseUpgradeResult(params("/ws?upgraded=card"))).toBe(null);
    expect(parseUpgradeResult(params("/ws"))).toBe(null);
  });
});

describe("checkoutReturnPath", () => {
  const base = { workspaceSlug: "ws", returnTo: "/ws/settings/team" };

  it("returns a new Compute plan to its origin with the result", () => {
    expect(
      checkoutReturnPath({ ...base, outcome: "subscribed", intent: "deploy", plan: "pro" }),
    ).toBe("/ws/settings/team?upgraded=compute&plan=pro");
  });

  it("hands Compute back to projects when checkout only saved a card", () => {
    expect(
      checkoutReturnPath({
        ...base,
        outcome: "none",
        intent: "deploy",
        plan: "pro",
        from: "billing",
      }),
    ).toBe("/ws/projects?pendingPlan=pro&from=billing");
  });

  it("hands Compute back to projects without a return path", () => {
    expect(
      checkoutReturnPath({
        workspaceSlug: "ws",
        outcome: "subscribed",
        intent: "deploy",
        plan: "pro",
        from: "create",
      }),
    ).toBe("/ws/projects?pendingPlan=pro&from=create");
  });

  it("returns a new API plan to its origin with the result", () => {
    expect(checkoutReturnPath({ ...base, outcome: "subscribed", intent: "api-subscription" })).toBe(
      "/ws/settings/team?upgraded=api",
    );
  });

  it("drops a foreign return path", () => {
    expect(
      checkoutReturnPath({
        workspaceSlug: "ws",
        returnTo: "//evil.com",
        outcome: "subscribed",
        intent: "api-subscription",
      }),
    ).toBe("/ws/settings/billing?upgraded=api");
  });

  it("claims no API upgrade when nothing was subscribed", () => {
    expect(checkoutReturnPath({ ...base, outcome: "none", intent: "api-subscription" })).toBe(
      "/ws/settings/billing",
    );
  });

  it("hands a gate checkout back to projects even with a return path", () => {
    expect(
      checkoutReturnPath({
        ...base,
        outcome: "subscribed",
        intent: "deploy",
        plan: "pro",
        from: "create",
      }),
    ).toBe("/ws/projects?pendingPlan=pro&from=create");
  });

  it("drops an unknown plan or origin from the projects hand-off", () => {
    expect(
      checkoutReturnPath({
        workspaceSlug: "ws",
        outcome: "none",
        intent: "deploy",
        plan: "pro&from=create",
        from: "evil",
      }),
    ).toBe("/ws/projects");
  });

  it("returns a deploy-gate checkout to the page it started on", () => {
    expect(
      checkoutReturnPath({
        workspaceSlug: "ws",
        returnTo: "/ws/projects/proj_1/deployments",
        outcome: "subscribed",
        intent: "deploy",
        plan: "starter",
        from: "deploy",
      }),
    ).toBe("/ws/projects/proj_1/deployments?upgraded=compute&plan=starter");
  });
});
