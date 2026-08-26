import { permissionValidation } from "@unkey/rbac";
import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import {
  ACTIONS,
  ALL_INSTANCES,
  RESOURCE_SCOPES,
  instancePath,
  resolveInstance,
} from "./catalogue.types";
import { buildUrn } from "./urn";

// Every action the Go backend can require of a root key today: the `ActionType`
// constants in pkg/rbac/permissions.go plus the `String()` methods in
// pkg/rbac/permissions/*.go. Keep in step with those by hand; this suite must
// not read Go at runtime.
const GO_ACTIONS = [
  "add_permission_to_key",
  "add_permission_to_role",
  "add_role_to_key",
  "connect_repository",
  "create_api",
  "create_app",
  "create_deployment",
  "create_domain",
  "create_environment",
  "create_identity",
  "create_key",
  "create_namespace",
  "create_permission",
  "create_portal",
  "create_portal_session",
  "create_project",
  "create_role",
  "create_variable",
  "decrypt_key",
  "delete_api",
  "delete_app",
  "delete_domain",
  "delete_identity",
  "delete_key",
  "delete_namespace",
  "delete_override",
  "delete_permission",
  "delete_portal",
  "delete_project",
  "delete_role",
  "encrypt_key",
  "generate_upload_url",
  "install_github",
  "limit",
  "list_overrides",
  "promote_deployment",
  "read_analytics",
  "read_api",
  "read_app",
  "read_deployment",
  "read_domain",
  "read_environment",
  "read_environment_variables",
  "read_gateway_requests",
  "read_identity",
  "read_key",
  "read_keyspace",
  "read_namespace",
  "read_override",
  "read_permission",
  "read_policies",
  "read_policy",
  "read_portal",
  "read_project",
  "read_role",
  "read_runtime_logs",
  "remove_environment_variables",
  "remove_permission_from_key",
  "remove_permission_from_role",
  "remove_role_from_key",
  "rollback_deployment",
  "set_environment_variables",
  "set_override",
  "set_policies",
  "start_deployment",
  "stop_deployment",
  "update_api",
  "update_app",
  "update_environment",
  "update_identity",
  "update_key",
  "update_namespace",
  "update_permission",
  "update_policy",
  "update_portal",
  "update_project",
  "update_role",
  "verify_domain",
  "verify_key",
  "write_policy",
];

// The model landed on the dashboard before the API caught up, so most of the
// vocabulary names a privilege Go cannot require yet: every `write_<resource>`
// (Go still splits create from update), every `/logs` read, the namespace limit
// and the GitHub app. Shorten this list as pkg/rbac grows.
const KNOWN_UNENFORCEABLE = [
  "delete_deployment",
  "delete_environment",
  "delete_environment_variable",
  "delete_gateway_policy",
  "delete_github_app",
  "delete_keyspace",
  "delete_ratelimit_namespace",
  "delete_ratelimit_override",
  "limit_ratelimit_namespace",
  "read_deployment_logs",
  "read_environment_variable",
  "read_gateway_logs",
  "read_gateway_policy",
  "read_github_app",
  "read_keyspace_logs",
  "read_ratelimit_logs",
  "read_ratelimit_namespace",
  "read_ratelimit_override",
  "write_app",
  "write_deployment",
  "write_domain",
  "write_environment",
  "write_environment_variable",
  "write_gateway_policy",
  "write_github_app",
  "write_identity",
  "write_key",
  "write_keyspace",
  "write_permission",
  "write_project",
  "write_ratelimit_namespace",
  "write_ratelimit_override",
  "write_role",
];

function everyGrant() {
  return RESOURCE_SCOPES.flatMap((scope) => {
    const catalogue = CATALOGUES[scope];
    const instance = resolveInstance(catalogue, ALL_INSTANCES);
    return catalogueRows(catalogue).flatMap((row) =>
      ACTIONS.flatMap((action) =>
        row.actions[action].map((grant) => ({
          name: grant.name,
          path: instancePath(grant.path, instance),
        })),
      ),
    );
  });
}

describe("catalogue grants", () => {
  it("only names permissions the catalog in @unkey/rbac knows", () => {
    const rejected = everyGrant()
      .map((grant) => buildUrn("ws_1234abcd", grant.path, grant.name))
      .filter((urn) => !permissionValidation.safeParse(urn).success);

    expect(rejected).toEqual([]);
  });

  it("names the actions the backend cannot enforce yet, and no others", () => {
    const backend = new Set(GO_ACTIONS);
    const emitted = new Set(everyGrant().map((grant) => grant.name));

    expect([...emitted].filter((name) => !backend.has(name)).sort()).toEqual(KNOWN_UNENFORCEABLE);
  });
});
