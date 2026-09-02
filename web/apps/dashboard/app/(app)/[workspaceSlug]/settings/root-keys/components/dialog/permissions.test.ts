import {
  apiActions,
  appActions,
  environmentActions,
  identityActions,
  portalActions,
  projectActions,
  ratelimitActions,
  rbacActions,
  workspaceActions,
} from "@unkey/rbac";
import { describe, expect, it } from "vitest";
import { appPermissions, projectPermissions, workspacePermissions } from "./permissions";
import { filterPermissionList, getAllPermissionNames } from "./utils/permissions";

const actionsByResource: Record<string, readonly string[]> = {
  api: apiActions.options,
  app: appActions.options,
  environment: environmentActions.options,
  identity: identityActions.options,
  portal: portalActions.options,
  project: projectActions.options,
  ratelimit: ratelimitActions.options,
  rbac: rbacActions.options,
  workspace: workspaceActions.options,
};

describe("workspacePermissions", () => {
  const listedPermissions = new Set<string>(getAllPermissionNames(workspacePermissions));

  for (const [resource, actions] of Object.entries(actionsByResource)) {
    it(`lists every ${resource} action`, () => {
      const missingPermissions = actions
        .map((action) => `${resource}.*.${action}`)
        .filter((permission) => !listedPermissions.has(permission));

      expect(missingPermissions).toEqual([]);
    });
  }

  it("shows role assignment permissions in search results", () => {
    expect(
      getAllPermissionNames(filterPermissionList(workspacePermissions, "permission_to_role")),
    ).toEqual(["rbac.*.add_permission_to_role"]);
    expect(
      getAllPermissionNames(
        filterPermissionList(workspacePermissions, "remove_permission_from_role"),
      ),
    ).toEqual(["rbac.*.remove_permission_from_role"]);
  });
});

describe("resource permissions", () => {
  it("lists actions supported for one project", () => {
    expect(getAllPermissionNames(projectPermissions("proj_12345678"))).toEqual([
      "project.proj_12345678.read_project",
      "project.proj_12345678.update_project",
      "project.proj_12345678.delete_project",
      "project.proj_12345678.create_app",
      "project.proj_12345678.create_deployment",
      "project.proj_12345678.read_deployment",
    ]);
  });

  it("lists every action for one app", () => {
    expect(getAllPermissionNames(appPermissions("app_12345678"))).toEqual(
      appActions.options.map((action) => `app.app_12345678.${action}`),
    );
  });
});
