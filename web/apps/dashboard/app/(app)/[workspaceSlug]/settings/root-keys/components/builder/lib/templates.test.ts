import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { offeredActions } from "./catalogue.types";
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
    expect(urnsOf("read")).toEqual([
      "unkey:v1:ws_123:projects/*#read_project",
      "unkey:v1:ws_123:projects/*/apps/*#read_app",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#read_environment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/variables/*#read_environment_variable",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/domains/*#read_domain",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#read_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*/logs#read_deployment_logs",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/gateway/logs#read_gateway_logs",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/gateway/policies/*#read_gateway_policy",
      "unkey:v1:ws_123:projects/*/keyspaces/*#read_keyspace",
      "unkey:v1:ws_123:projects/*/keyspaces/*/logs#read_keyspace_logs",
      "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#read_key",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*#read_ratelimit_namespace",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/logs#read_ratelimit_logs",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#read_ratelimit_override",
      "unkey:v1:ws_123:projects/*/identities/*#read_identity",
      "unkey:v1:ws_123:projects/*/rbac/roles/*#read_role",
      "unkey:v1:ws_123:projects/*/rbac/permissions/*#read_permission",
      "unkey:v1:ws_123:github/apps/*#read_github_app",
    ]);
  });

  it("materialises All write permissions into every action of every row", () => {
    expect(materialise("write")).toHaveLength(1);
    const offered = catalogueRows(CATALOGUES.workspace).reduce(
      (total, row) => total + offeredActions(row).length,
      0,
    );
    expect(urnsOf("write")).toHaveLength(offered);
  });

  it("gives full control the narrow key actions too", () => {
    expect(urnsOf("write")).toContain("unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#decrypt_key");
    expect(urnsOf("write")).toContain("unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#verify_key");
    expect(materialise("write")[0].selection.key).toEqual([
      "read",
      "write",
      "delete",
      "verify",
      "decrypt",
    ]);
  });

  it("keeps decrypt out of every template but full control", () => {
    for (const template of TEMPLATES) {
      const decrypts = buildUrns(ws, template.materialise()).some((urn) => urn.includes("decrypt"));
      expect(decrypts).toBe(template.id === "write");
    }
  });

  it("materialises Verify keys into 1 policy", () => {
    expect(materialise("verify")).toHaveLength(1);
    expect(urnsOf("verify")).toEqual(["unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#verify_key"]);
  });

  it("materialises Standalone ratelimiting into 1 policy", () => {
    expect(materialise("ratelimit")).toHaveLength(1);
    expect(urnsOf("ratelimit")).toEqual([
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*#limit_ratelimit_namespace",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/logs#read_ratelimit_logs",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#read_ratelimit_override",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#write_ratelimit_override",
      "unkey:v1:ws_123:projects/*/ratelimits/namespaces/*/overrides/*#delete_ratelimit_override",
    ]);
  });

  it("materialises Start new into 1 policy", () => {
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
