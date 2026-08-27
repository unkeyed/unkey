import { describe, expect, it } from "vitest";
import { type Policy, isPolicyComplete } from "./policy";
import { type RootKeyTemplate, TEMPLATES, type TemplateId } from "./templates";
import { buildUrns } from "./urn";

const ws = "ws_123";

const templateOf = (id: TemplateId): RootKeyTemplate => {
  const template = TEMPLATES.find((entry) => entry.id === id);
  if (!template) {
    throw new Error(`no template ${id}`);
  }
  return template;
};

const materialise = (id: TemplateId): Policy[] => templateOf(id).materialise();

const urnsOf = (id: TemplateId): string[] => buildUrns(ws, materialise(id));

describe("TEMPLATES", () => {
  it("materialises All read permissions into one workspace policy", () => {
    expect(materialise("read")).toHaveLength(1);
    expect(urnsOf("read")).toContain("unkey:v1:ws_123:github/apps/*#read_github_app");
    expect(urnsOf("read")).toContain(
      "unkey:v1:ws_123:projects/*/keyspaces/*/logs#read_keyspace_logs",
    );
    expect(urnsOf("read")).toContain(
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/logs#read_ratelimit_logs",
    );
    expect(urnsOf("read").every((urn) => /#read_[a-z_]+$/.test(urn))).toBe(true);
  });

  it("materialises All write permissions into one full-control workspace policy", () => {
    expect(materialise("write")).toHaveLength(1);
    expect(urnsOf("write")).toContain(
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#write_deployment",
    );
    expect(urnsOf("write")).not.toContain(
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#start_deployment",
    );
    expect(urnsOf("write")).not.toContain(
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#stop_deployment",
    );
    expect(urnsOf("write")).toContain("unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#decrypt_key");
    expect(urnsOf("write")).toContain(
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*#limit_ratelimit_namespace",
    );
  });

  it("materialises Verify keys into verify permissions only", () => {
    expect(materialise("verify")).toHaveLength(1);
    expect(urnsOf("verify")).toEqual(["unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#verify_key"]);
  });

  it("materialises Standalone ratelimiting into namespace limit and override management", () => {
    expect(materialise("ratelimit")).toHaveLength(1);
    expect(urnsOf("ratelimit")).toEqual([
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*#limit_ratelimit_namespace",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/logs#read_ratelimit_logs",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#read_ratelimit_override",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#write_ratelimit_override",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#delete_ratelimit_override",
    ]);
  });

  it("materialises Start new into 1 empty policy", () => {
    expect(materialise("custom")).toHaveLength(1);
    expect(urnsOf("custom")).toEqual([]);
  });

  it("keeps the gallery copy", () => {
    expect(
      TEMPLATES.map((template) => [template.id, template.title, template.description]),
    ).toEqual([
      ["read", "All read permissions", "Every resource, read only"],
      ["write", "All write permissions", "Every resource, full control"],
      ["verify", "Verify keys", "Every keyspace, verify only"],
      ["ratelimit", "Standalone ratelimiting", "Namespaces, logs, and overrides"],
      ["custom", "Start new", "Build a custom policy"],
    ]);
  });

  it("materialises policies that are valid on arrival, except the custom tile", () => {
    for (const template of TEMPLATES) {
      const complete = template.materialise().every(isPolicyComplete);
      expect(complete).toBe(template.id !== "custom");
    }
  });
});
