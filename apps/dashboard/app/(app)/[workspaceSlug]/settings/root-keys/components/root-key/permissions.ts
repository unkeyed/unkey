import type { UnkeyPermission } from "@unkey/rbac";

export type UnkeyPermissions = {
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
    read_analytics: {
      description: "Query analytics data for all API's in this workspace using SQL.",
      permission: "api.*.read_analytics",
    },
  },
  Keys: {
    verify_key: {
      description: "Verify API keys and enforce rate limits and permissions.",
      permission: "api.*.verify_key",
    },
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
    encrypt_key: {
      description: "Encrypt keys in this workspace",
      permission: "api.*.encrypt_key",
    },
    decrypt_key: {
      description: "Decrypt keys in this workspace",
      permission: "api.*.decrypt_key",
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
    set_override: {
      description: "Set a ratelimit override for an identifier.",
      permission: "ratelimit.*.set_override",
    },
    read_override: {
      description: "read ratelimit override for an identifier.",
      permission: "ratelimit.*.read_override",
    },
    delete_override: {
      description: "Delete ratelimit override for an identifier.",
      permission: "ratelimit.*.delete_override",
    },
  },
  Permissions: {
    create_role: {
      description: "Create a new role in this workspace",
      permission: "rbac.*.create_role",
    },
    read_role: {
      description: "Read roles in this workspace",
      permission: "rbac.*.read_role",
    },
    delete_role: {
      description: "Delete a role in this workspace",
      permission: "rbac.*.delete_role",
    },
    create_permission: {
      description: "Create a new permission in this workspace",
      permission: "rbac.*.create_permission",
    },
    read_permission: {
      description: "Read permissions in this workspace",
      permission: "rbac.*.read_permission",
    },
    delete_permission: {
      description: "Delete a permission in this workspace",
      permission: "rbac.*.delete_permission",
    },
    add_permission_to_key: {
      description: "Add a permission to a key",
      permission: "rbac.*.add_permission_to_key",
    },
    remove_permission_from_key: {
      description: "Remove a permission from a key",
      permission: "rbac.*.remove_permission_from_key",
    },
    add_role_to_key: {
      description: "Add a role to a key",
      permission: "rbac.*.add_role_to_key",
    },
    remove_role_from_key: {
      description: "Remove a role from a key",
      permission: "rbac.*.remove_role_from_key",
    },
  },
  Identities: {
    create_identity: {
      description: "Create a new identity in this workspace",
      permission: "identity.*.create_identity",
    },
    read_identity: {
      description: "Read an identity in this workspace",
      permission: "identity.*.read_identity",
    },
    update_identity: {
      description: "Update an identity in this workspace",
      permission: "identity.*.update_identity",
    },
    delete_identity: {
      description: "Delete an identity in this workspace",
      permission: "identity.*.delete_identity",
    },
  },
} satisfies Record<string, UnkeyPermissions>;

export function apiPermissions(apiId: string): {
  [category: string]: UnkeyPermissions;
} {
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
      read_analytics: {
        description: "Query analytics data for this API using SQL.",
        permission: `api.${apiId}.read_analytics`,
      },
    },
    Keys: {
      verify_key: {
        description:
          "Verify keys belonging to this API and enforce their rate limits and permissions.",
        permission: `api.${apiId}.verify_key`,
      },
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
      encrypt_key: {
        description: "Encrypt keys belonging to this API",
        permission: `api.${apiId}.encrypt_key`,
      },
      decrypt_key: {
        description: "Decrypt keys belonging to this API",
        permission: `api.${apiId}.decrypt_key`,
      },
    },
  };
}
