import { INSTANCE_TOKEN, type ScopeCatalogue } from "./catalogue.types";

const KEYSPACE_PATH = `keyspaces/${INSTANCE_TOKEN}`;

export const keyspacesCatalogue: ScopeCatalogue = {
  scope: "keyspaces",
  label: "Keyspaces",
  allLabel: "All keyspaces",
  instanceNoun: "keyspaces",
  groups: [
    {
      id: "keyspace",
      label: "Keyspace",
      rows: [
        {
          id: "keyspace",
          label: "Keyspace settings",
          path: KEYSPACE_PATH,
          resource: "keyspace",
          actions: {
            read: [{ name: "read_keyspace" }],
            write: [{ name: "update_keyspace" }],
            delete: [{ name: "delete_keyspace" }],
          },
        },
        {
          id: "key",
          label: "API keys",
          path: `${KEYSPACE_PATH}/keys/*`,
          resource: "key",
          actions: {
            read: [{ name: "read_key" }, { name: "verify_key" }],
            write: [{ name: "create_key", path: KEYSPACE_PATH }, { name: "update_key" }],
            delete: [{ name: "delete_key" }],
          },
        },
      ],
    },
  ],
};
