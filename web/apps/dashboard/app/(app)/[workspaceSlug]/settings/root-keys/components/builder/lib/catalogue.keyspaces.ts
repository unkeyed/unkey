import { INSTANCE_TOKEN, type ScopeCatalogue, permissionRow } from "./catalogue.types";

const KEYSPACE_PATH = `keyspaces/${INSTANCE_TOKEN}`;

export const keyspacesCatalogue: ScopeCatalogue = {
  scope: "keyspaces",
  label: "Keyspaces",
  allLabel: "All keyspaces",
  instanceNoun: "keyspaces",
  groups: [
    {
      id: "keyspaces",
      label: "Keyspaces",
      rows: [
        permissionRow({
          id: "keyspace",
          label: "Keyspace settings",
          path: KEYSPACE_PATH,
          resource: "keyspace",
          actions: {
            write: [{ name: "update_keyspace" }],
          },
        }),
        permissionRow({
          id: "key",
          label: "API keys",
          path: `${KEYSPACE_PATH}/keys/*`,
          resource: "key",
          actions: {
            read: [{ name: "read_key" }, { name: "verify_key" }],
            write: [{ name: "create_key", path: KEYSPACE_PATH }, { name: "update_key" }],
            decrypt: [{ name: "decrypt_key" }],
          },
        }),
      ],
    },
  ],
};
