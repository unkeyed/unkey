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
          label: "End-user identities",
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
          label: "Role definitions",
          path: "rbac/roles/*",
          resource: "role",
        },
        {
          id: "permission",
          label: "Permission definitions",
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
          label: "Encryption keys",
          path: "vault/keys/*",
          resource: "vault_key",
        },
      ],
    },
  ],
};
