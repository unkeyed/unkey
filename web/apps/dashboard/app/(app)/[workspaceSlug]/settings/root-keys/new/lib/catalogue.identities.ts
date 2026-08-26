import { type ScopeCatalogue, actionGrant } from "./catalogue.types";

export const identitiesCatalogue: ScopeCatalogue = {
  scope: "identities",
  label: "All identities",
  allLabel: "All identities",
  instanceNoun: null,
  groups: [
    {
      id: "identities",
      label: "Identities",
      rows: [
        {
          id: "identity",
          label: "Identities",
          path: "projects/*/identities/*",
          resource: "identity",
          actions: {
            read_identity: actionGrant("read_identity", "identities:read"),
            write_identity: actionGrant("write_identity", "identities:write"),
            delete_identity: actionGrant("delete_identity", "identities:delete"),
          },
        },
      ],
    },
  ],
};
