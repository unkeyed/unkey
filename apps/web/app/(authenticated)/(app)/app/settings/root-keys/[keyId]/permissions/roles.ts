import { Role } from "@unkey/rbac";

type Roles = {
  [action: string]: {
    description: string;
    role: Role;
  };
};

export const workspaceRoles = {
  create_api: {
    description: "Create new apis in this workspace.",
    role: "api.*.create_api",
  },
  read_api: {
    description: "Read information about any existing or future API in this workspace.",
    role: "api.*.read_api",
  },
  update_api: {
    description: "Update any existing or future API in this workspace.",
    role: "api.*.update_api",
  },
  delete_api: {
    description: "Delete all apis in this workspace.",
    role: "api.*.delete_api",
  },
} satisfies Roles;

export function apiRoles(apiId: string): { [category: string]: Roles } {
  return {
    API: {
      read_api: {
        description: "Read information about this API.",
        role: `api.${apiId}.read_api`,
      },
      delete_api: {
        description:
          "Delete this API. Enabling this role, does not grant access to delete other APIs in this workspace.",
        role: `api.${apiId}.delete_api`,
      },
      update_api: {
        description: "Update this API.",
        role: `api.${apiId}.update_api`,
      },
    },
    Keys: {
      create_key: {
        description: "Create a new key for this API.",
        role: `api.${apiId}.create_key`,
      },
      read_key: {
        description: "Read information and analytics about keys.",
        role: `api.${apiId}.read_key`,
      },
      update_key: {
        description: "Update limits or other information about a key.",
        role: `api.${apiId}.update_key`,
      },
      delete_key: {
        description: "Delete keys belonging to this API.",
        role: `api.${apiId}.delete_key`,
      },
    },
  };
}
