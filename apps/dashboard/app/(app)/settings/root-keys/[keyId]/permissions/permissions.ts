import type { UnkeyPermission } from "@unkey/rbac";

type UnkeyPermissions = {
  [action: string]: {
    description: string;
    permission: UnkeyPermission;
  };
};

export const workspacePermissions = {
  API: {
    create_api: {
      description: "Create new apis in this workspace.",
      permission: "api.*.create_api",
    },
    read_api: {
      description: "Read information about any existing or future API in this workspace.",
      permission: "api.*.read_api",
    },
    update_api: {
      description: "Update any existing or future API in this workspace.",
      permission: "api.*.update_api",
    },
    delete_api: {
      description: "Delete apis in this workspace.",
      permission: "api.*.delete_api",
    },
  },
  Keys: {
    create_key: {
      description: "Create new keys in this workspace.",
      permission: "api.*.create_key",
    },
    read_key: {
      description: "Read information about any existing or future key in this workspace.",
      permission: "api.*.read_key",
    },
    update_key: {
      description: "Update any existing or future key in this workspace.",
      permission: "api.*.update_key",
    },
    delete_key: {
      description: "Delete keys in this workspace.",
      permission: "api.*.delete_key",
    },
  },
  Ratelimit: {
    create_namespace: {
      description: "Create new namespaces in this workspace.",
      permission: "ratelimit.*.create_namespace",
    },
    read_namespace: {
      description: "Read information about any existing or future namespace in this workspace.",
      permission: "ratelimit.*.read_namespace",
    },
    limit: {
      description: "Limit an identifier",
      permission: "ratelimit.*.limit",
    },
    update_namespace: {
      description: "Update any existing or future namespace in this workspace.",
      permission: "ratelimit.*.update_namespace",
    },
    delete_namespace: {
      description: "Delete namespaces in this workspace.",
      permission: "ratelimit.*.delete_namespace",
    },
  },
} satisfies Record<string, UnkeyPermissions>;

export function apiPermissions(apiId: string): { [category: string]: UnkeyPermissions } {
  return {
    API: {
      read_api: {
        description: "Read information about this API.",
        permission: `api.${apiId}.read_api`,
      },
      delete_api: {
        description:
          "Delete this API. Enabling this permission, does not grant access to delete other APIs in this workspace.",
        permission: `api.${apiId}.delete_api`,
      },
      update_api: {
        description: "Update this API.",
        permission: `api.${apiId}.update_api`,
      },
    },
    Keys: {
      create_key: {
        description: "Create a new key for this API.",
        permission: `api.${apiId}.create_key`,
      },
      read_key: {
        description: "Read information and analytics about keys.",
        permission: `api.${apiId}.read_key`,
      },
      update_key: {
        description: "Update limits or other information about a key.",
        permission: `api.${apiId}.update_key`,
      },
      delete_key: {
        description: "Delete keys belonging to this API.",
        permission: `api.${apiId}.delete_key`,
      },
    },
  };
}
