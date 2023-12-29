import { Role } from "@unkey/rbac";

type Roles = {
  [action: string]: {
    description: string;
    role: Role | ((id: string) => Role);
  };
};

export const workspaceRoles = {
  read_api: {
    description: "Read information about any existing or future API in this workspace.",
    role: "api.*.read_api",
  },
  create_api: {
    description:
      "Create new apis in this workspace. Unless this key has global access, it will not be able to read or modify the API after creation.",
    role: "api.*.create_api",
  },
} satisfies Roles;

export const apiRoles = {
  read_api: {
    description: "Read information about an API",
    role: (apiId: string) => `api.${apiId}.read_api`,
  },
  create_api: {
    description: "Create new apis in this workspace",
    role: (apiId: string) => `api.${apiId}.create_api`,
  },
  delete_api: {
    description: "Delete an API",
    role: (apiId: string) => `api.${apiId}.delete_api`,
  },
  update_api: {
    description: "Update an API",
    role: (apiId: string) => `api.${apiId}.update_api`,
  },

  create_key: {
    description: "Create a new key for this API",
    role: (apiId: string) => `api.${apiId}.create_key`,
  },
  update_key: {
    description: "Update limits or other information about a key",
    role: (apiId: string) => `api.${apiId}.update_key`,
  },
  delete_key: {
    description: "Delete a key",
    role: (apiId: string) => `api.${apiId}.delete_key`,
  },
  read_key: {
    description: "Read information and analytics about a key",
    role: (apiId: string) => `api.${apiId}.read_key`,
  },
} satisfies Roles;
