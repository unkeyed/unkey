import { describe, expect, it } from "vitest";
import { TEMPLATES, type TemplateId } from "../builder/lib/templates";
import { buildUrns } from "../builder/lib/urn";
import { permissionSummary } from "./permission-summary";

const ws = "ws_123";

function grantsOf(id: TemplateId): string[] {
  const template = TEMPLATES.find((entry) => entry.id === id);
  if (!template) {
    throw new Error(`no template ${id}`);
  }
  return buildUrns(ws, template.materialise());
}

describe("permissionSummary", () => {
  it("reads none for a key without grants", () => {
    expect(permissionSummary(ws, [])).toEqual({ type: "none" });
  });

  it.each(["read", "write", "verify", "ratelimit"] as const)(
    "names the %s template when the grants match it exactly",
    (id) => {
      expect(permissionSummary(ws, grantsOf(id))).toEqual({ type: "template", template: id });
    },
  );

  it("ignores grant order and duplicates", () => {
    const grants = grantsOf("verify");
    expect(permissionSummary(ws, [...grants, ...grants].reverse())).toEqual({
      type: "template",
      template: "verify",
    });
  });

  it("reads restricted for a subset of a template", () => {
    expect(permissionSummary(ws, grantsOf("read").slice(1))).toEqual({ type: "restricted" });
  });

  it("reads restricted for a template plus an extra grant", () => {
    expect(
      permissionSummary(ws, [...grantsOf("verify"), "unkey:v1:ws_123:projects/*#read"]),
    ).toEqual({ type: "restricted" });
  });

  it("reads restricted for another workspace's template grants", () => {
    expect(permissionSummary("ws_other", grantsOf("write"))).toEqual({ type: "restricted" });
  });

  it("reads restricted for legacy permission names", () => {
    expect(permissionSummary(ws, ["api.*.verify_key"])).toEqual({ type: "restricted" });
  });
});
