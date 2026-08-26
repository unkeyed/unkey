import { type ScopeCatalogue, permissionRow } from "./catalogue.types";

export const identitiesCatalogue: ScopeCatalogue = {
  scope: "identities",
  label: "Identities",
  allLabel: "All identities",
  instanceNoun: null,
  groups: [
    {
      id: "identities",
      label: "Identities",
      rows: [
        permissionRow({
          id: "identity",
          label: "End-user identities",
          path: "identities/*",
          resource: "identity",
        }),
      ],
    },
  ],
};
