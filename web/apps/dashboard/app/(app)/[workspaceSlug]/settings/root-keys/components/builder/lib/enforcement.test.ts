import {
  apiActions,
  appActions,
  environmentActions,
  identityActions,
  projectActions,
  ratelimitActions,
  rbacActions,
} from "@unkey/rbac";
import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { ACTIONS, RESOURCE_SCOPES } from "./catalogue.types";

// Copied from the `String()` methods in pkg/rbac/permissions/*.go. Keep in step
// with that package by hand; this suite must not read Go at runtime.
const GO_ACTIONS = [
  "create_app",
  "create_deployment",
  "create_domain",
  "create_environment",
  "create_identity",
  "create_key",
  "create_permission",
  "create_portal",
  "create_portal_session",
  "create_project",
  "create_variable",
  "decrypt_key",
  "delete_app",
  "delete_domain",
  "delete_identity",
  "delete_key",
  "delete_override",
  "delete_portal",
  "delete_project",
  "encrypt_key",
  "promote_deployment",
  "read_domain",
  "read_environment",
  "read_environment_variables",
  "read_identity",
  "read_key",
  "read_keyspace",
  "read_override",
  "read_policy",
  "read_portal",
  "remove_environment_variables",
  "rollback_deployment",
  "set_environment_variables",
  "set_override",
  "start_deployment",
  "stop_deployment",
  "update_environment",
  "update_identity",
  "update_key",
  "update_portal",
  "update_project",
  "verify_domain",
  "verify_key",
  "write_policy",
];

const BACKEND_ACTIONS = new Set([
  ...GO_ACTIONS,
  ...apiActions.options,
  ...appActions.options,
  ...environmentActions.options,
  ...identityActions.options,
  ...projectActions.options,
  ...ratelimitActions.options,
  ...rbacActions.options,
]);

const KNOWN_UNENFORCEABLE = [
  "delete_deployment",
  "delete_environment",
  "delete_keyspace",
  "update_keyspace",
];

describe("catalogue action names", () => {
  it("names four actions the backend cannot enforce yet, and no others", () => {
    const emitted = new Set(
      RESOURCE_SCOPES.flatMap((scope) =>
        catalogueRows(CATALOGUES[scope]).flatMap((row) =>
          ACTIONS.flatMap((action) => row.actions[action].map((grant) => grant.name)),
        ),
      ),
    );
    expect([...emitted].filter((name) => !BACKEND_ACTIONS.has(name)).sort()).toEqual(
      KNOWN_UNENFORCEABLE,
    );
  });
});
