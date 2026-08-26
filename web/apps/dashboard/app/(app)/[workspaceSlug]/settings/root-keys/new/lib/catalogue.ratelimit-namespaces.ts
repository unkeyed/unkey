import { INSTANCE_TOKEN, type ScopeCatalogue, actionGrant } from "./catalogue.types";

const NAMESPACE_PATH = INSTANCE_TOKEN;
const NAMESPACE_ALL_PATH = "projects/*/ratelimits/namespaces/*";

export const ratelimitNamespacesCatalogue: ScopeCatalogue = {
  scope: "ratelimit-namespaces",
  label: "Specific rate limit namespaces",
  allLabel: "All namespaces",
  instanceNoun: "namespaces",
  groups: [
    {
      id: "namespace",
      label: "Rate limiting",
      rows: [
        {
          id: "ratelimit_namespace",
          label: "Rate limit namespaces",
          path: NAMESPACE_PATH,
          allPath: NAMESPACE_ALL_PATH,
          resource: "ratelimit_namespace",
          actions: {
            read_ratelimit_namespace: actionGrant(
              "read_ratelimit_namespace",
              "ratelimit_namespaces:read",
            ),
            write_ratelimit_namespace: actionGrant(
              "write_ratelimit_namespace",
              "ratelimit_namespaces:write",
            ),
            delete_ratelimit_namespace: actionGrant(
              "delete_ratelimit_namespace",
              "ratelimit_namespaces:delete",
            ),
            limit_ratelimit_namespace: actionGrant(
              "limit_ratelimit_namespace",
              "ratelimit_namespaces:limit",
            ),
          },
        },
        {
          id: "ratelimit_log",
          label: "Logs",
          path: `${NAMESPACE_PATH}/logs`,
          allPath: `${NAMESPACE_ALL_PATH}/logs`,
          resource: "ratelimit_log",
          actions: {
            read_ratelimit_logs: actionGrant("read_ratelimit_logs", "ratelimit_logs:read"),
          },
        },
        {
          id: "ratelimit_override",
          label: "Rate limit overrides",
          path: `${NAMESPACE_PATH}/overrides/*`,
          allPath: `${NAMESPACE_ALL_PATH}/overrides/*`,
          resource: "ratelimit_override",
          actions: {
            read_ratelimit_override: actionGrant(
              "read_ratelimit_override",
              "ratelimit_overrides:read",
            ),
            write_ratelimit_override: actionGrant(
              "write_ratelimit_override",
              "ratelimit_overrides:write",
            ),
            delete_ratelimit_override: actionGrant(
              "delete_ratelimit_override",
              "ratelimit_overrides:delete",
            ),
          },
        },
      ],
    },
  ],
};
