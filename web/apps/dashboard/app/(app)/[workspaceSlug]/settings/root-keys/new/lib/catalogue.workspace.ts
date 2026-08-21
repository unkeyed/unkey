import type { ScopeCatalogue } from "./catalogue.types";

export const workspaceCatalogue: ScopeCatalogue = {
  scope: "workspace",
  label: "Workspace",
  allLabel: "Everything in this workspace",
  instanceNoun: null,
  groups: [
    {
      id: "identities",
      label: "Identities",
      rows: [
        {
          id: "identity",
          label: "Identities",
          description: "End-user identities and their metadata.",
          path: "identities/*",
          resource: "identity",
        },
      ],
    },
    {
      id: "rbac",
      label: "RBAC",
      rows: [
        {
          id: "role",
          label: "Roles",
          description: "Role definitions in this workspace.",
          path: "rbac/roles/*",
          resource: "role",
        },
        {
          id: "permission",
          label: "Permissions",
          description: "Permission definitions in this workspace.",
          path: "rbac/permissions/*",
          resource: "permission",
        },
      ],
    },
    {
      id: "vault",
      label: "Vault",
      rows: [
        {
          id: "vault_key",
          label: "Vault keys",
          description: "Encryption keys held by vault.",
          path: "vault/keys/*",
          resource: "vault_key",
        },
      ],
    },
  ],
};
