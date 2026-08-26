import { INSTANCE_TOKEN, type ScopeCatalogue, actionGrant } from "./catalogue.types";

const KEYSPACE_PATH = INSTANCE_TOKEN;
const KEYSPACE_ALL_PATH = "projects/*/keyspaces/*";

export const keyspacesCatalogue: ScopeCatalogue = {
  scope: "keyspaces",
  label: "Specific keyspaces",
  allLabel: "All keyspaces",
  instanceNoun: "keyspaces",
  groups: [
    {
      id: "keyspace",
      label: "Key management",
      rows: [
        {
          id: "keyspace",
          label: "Keyspaces",
          path: KEYSPACE_PATH,
          allPath: KEYSPACE_ALL_PATH,
          resource: "keyspace",
          actions: {
            read_keyspace: actionGrant("read_keyspace", "keyspaces:read"),
            write_keyspace: actionGrant("write_keyspace", "keyspaces:write"),
            delete_keyspace: actionGrant("delete_keyspace", "keyspaces:delete"),
          },
        },
        {
          id: "keyspace_log",
          label: "Logs",
          path: `${KEYSPACE_PATH}/logs`,
          allPath: `${KEYSPACE_ALL_PATH}/logs`,
          resource: "keyspace_log",
          actions: {
            read_keyspace_logs: actionGrant("read_keyspace_logs", "keyspace_logs:read"),
          },
        },
        {
          id: "key",
          label: "Keys",
          path: `${KEYSPACE_PATH}/keys/*`,
          allPath: `${KEYSPACE_ALL_PATH}/keys/*`,
          resource: "key",
          actions: {
            read_key: actionGrant("read_key", "keys:read"),
            write_key: actionGrant("write_key", "keys:write"),
            delete_key: actionGrant("delete_key", "keys:delete"),
            decrypt_key: actionGrant("decrypt_key", "keys:decrypt"),
            verify_key: actionGrant("verify_key", "keys:verify"),
          },
        },
      ],
    },
  ],
};
