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
    expect(urnsOf("read")).toEqual([
      "unkey:v1:ws_123:projects/*#read_project",
      "unkey:v1:ws_123:projects/*/apps/*#read_app",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#read_environment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#read_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/domains/*#read_domain",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#read_environment_variables",
      "unkey:v1:ws_123:keyspaces/*#read_keyspace",
      "unkey:v1:ws_123:keyspaces/*/keys/*#read_key",
      "unkey:v1:ws_123:keyspaces/*/keys/*#verify_key",
      "unkey:v1:ws_123:ratelimits/namespaces/*#read_namespace",
      "unkey:v1:ws_123:ratelimits/namespaces/*#limit",
      "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/*#read_override",
      "unkey:v1:ws_123:identities/*#read_identity",
      "unkey:v1:ws_123:rbac/roles/*#read_role",
      "unkey:v1:ws_123:rbac/permissions/*#read_permission",
      "unkey:v1:ws_123:vault/keys/*#read_vault_key",
    ]);
  });

  it("materialises All write permissions into one workspace policy", () => {
    expect(materialise("write")).toHaveLength(1);
    expect(urnsOf("write")).toEqual([
      "unkey:v1:ws_123:projects/*#read_project",
      "unkey:v1:ws_123:projects/*#update_project",
      "unkey:v1:ws_123:projects/*#delete_project",
      "unkey:v1:ws_123:projects/*/apps/*#read_app",
      "unkey:v1:ws_123:projects/*#create_app",
      "unkey:v1:ws_123:projects/*/apps/*#update_app",
      "unkey:v1:ws_123:projects/*/apps/*#connect_repository",
      "unkey:v1:ws_123:projects/*/apps/*#delete_app",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#read_environment",
      "unkey:v1:ws_123:projects/*/apps/*#create_environment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#update_environment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#delete_environment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#read_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#create_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#start_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#stop_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#promote_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#rollback_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/deployments/*#delete_deployment",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/domains/*#read_domain",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#create_domain",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#verify_domain",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*/domains/*#delete_domain",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#read_environment_variables",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#create_variable",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#set_environment_variables",
      "unkey:v1:ws_123:projects/*/apps/*/environments/*#remove_environment_variables",
      "unkey:v1:ws_123:keyspaces/*#read_keyspace",
      "unkey:v1:ws_123:keyspaces/*#update_keyspace",
      "unkey:v1:ws_123:keyspaces/*#delete_keyspace",
      "unkey:v1:ws_123:keyspaces/*/keys/*#read_key",
      "unkey:v1:ws_123:keyspaces/*/keys/*#verify_key",
      "unkey:v1:ws_123:keyspaces/*#create_key",
      "unkey:v1:ws_123:keyspaces/*/keys/*#update_key",
      "unkey:v1:ws_123:keyspaces/*/keys/*#delete_key",
      "unkey:v1:ws_123:ratelimits/namespaces/*#read_namespace",
      "unkey:v1:ws_123:ratelimits/namespaces/*#limit",
      "unkey:v1:ws_123:ratelimits/namespaces/*#update_namespace",
      "unkey:v1:ws_123:ratelimits/namespaces/*#delete_namespace",
      "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/*#read_override",
      "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/*#set_override",
      "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/*#delete_override",
      "unkey:v1:ws_123:identities/*#read_identity",
      "unkey:v1:ws_123:identities/*#create_identity",
      "unkey:v1:ws_123:identities/*#update_identity",
      "unkey:v1:ws_123:identities/*#delete_identity",
      "unkey:v1:ws_123:rbac/roles/*#read_role",
      "unkey:v1:ws_123:rbac/roles/*#create_role",
      "unkey:v1:ws_123:rbac/roles/*#update_role",
      "unkey:v1:ws_123:rbac/roles/*#delete_role",
      "unkey:v1:ws_123:rbac/permissions/*#read_permission",
      "unkey:v1:ws_123:rbac/permissions/*#create_permission",
      "unkey:v1:ws_123:rbac/permissions/*#update_permission",
      "unkey:v1:ws_123:rbac/permissions/*#delete_permission",
      "unkey:v1:ws_123:vault/keys/*#read_vault_key",
      "unkey:v1:ws_123:vault/keys/*#create_vault_key",
      "unkey:v1:ws_123:vault/keys/*#update_vault_key",
      "unkey:v1:ws_123:vault/keys/*#delete_vault_key",
    ]);
  });

  it("materialises Verify keys into 1 policy", () => {
    expect(materialise("verify")).toHaveLength(1);
    expect(urnsOf("verify")).toEqual([
      "unkey:v1:ws_123:keyspaces/*/keys/*#read_key",
      "unkey:v1:ws_123:keyspaces/*/keys/*#verify_key",
    ]);
  });

  it("materialises Standalone ratelimiting into 1 policy", () => {
    expect(materialise("ratelimit")).toHaveLength(1);
    expect(urnsOf("ratelimit")).toEqual([
      "unkey:v1:ws_123:ratelimits/namespaces/*#read_namespace",
      "unkey:v1:ws_123:ratelimits/namespaces/*#limit",
      "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/*#read_override",
      "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/*#set_override",
      "unkey:v1:ws_123:ratelimits/namespaces/*/overrides/*#delete_override",
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
      ["ratelimit", "Standalone ratelimiting", "Namespaces and overrides"],
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
