import type { ScopeCatalogue } from "./catalogue.types";

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
        {
          id: "identity",
          label: "End-user identities",
          path: "identities/*",
          resource: "identity",
        },
      ],
    },
  ],
};
