import type { ScopeCatalogue } from "./catalogue.types";

export const vaultCatalogue: ScopeCatalogue = {
  scope: "vault",
  label: "Vault",
  allLabel: "All encryption keys",
  instanceNoun: null,
  groups: [
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
