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
            read_ratelimit_namespace: actionGrant("read_ratelimit_namespace"),
            write_ratelimit_namespace: actionGrant("write_ratelimit_namespace"),
            delete_ratelimit_namespace: actionGrant("delete_ratelimit_namespace"),
            limit_ratelimit_namespace: actionGrant("limit_ratelimit_namespace"),
          },
        },
        {
          id: "ratelimit_log",
          label: "Logs",
          path: `${NAMESPACE_PATH}/logs`,
          allPath: `${NAMESPACE_ALL_PATH}/logs`,
          resource: "ratelimit_log",
          actions: {
            read_ratelimit_logs: actionGrant("read_ratelimit_logs"),
          },
        },
        {
          id: "ratelimit_override",
          label: "Rate limit overrides",
          path: `${NAMESPACE_PATH}/overrides/*`,
          allPath: `${NAMESPACE_ALL_PATH}/overrides/*`,
          resource: "ratelimit_override",
          actions: {
            read_ratelimit_override: actionGrant("read_ratelimit_override"),
            write_ratelimit_override: actionGrant("write_ratelimit_override"),
            delete_ratelimit_override: actionGrant("delete_ratelimit_override"),
          },
        },
      ],
    },
  ],
};
