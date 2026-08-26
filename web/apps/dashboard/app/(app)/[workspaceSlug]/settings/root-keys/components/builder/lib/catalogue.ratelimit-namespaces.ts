import { INSTANCE_TOKEN, type ScopeCatalogue, permissionRow } from "./catalogue.types";

const NAMESPACE_PATH = `ratelimits/namespaces/${INSTANCE_TOKEN}`;

export const ratelimitNamespacesCatalogue: ScopeCatalogue = {
  scope: "ratelimit-namespaces",
  label: "Ratelimit namespaces",
  allLabel: "All namespaces",
  instanceNoun: "namespaces",
  groups: [
    {
      id: "ratelimit-namespaces",
      label: "Ratelimit namespaces",
      rows: [
        permissionRow({
          id: "namespace",
          label: "Namespace settings",
          path: NAMESPACE_PATH,
          resource: "namespace",
          actions: {
            read: [{ name: "read_namespace" }, { name: "limit" }],
            write: [{ name: "update_namespace" }],
          },
        }),
        permissionRow({
          id: "override",
          label: "Limit overrides",
          path: `${NAMESPACE_PATH}/overrides/*`,
          resource: "override",
          actions: {
            write: [{ name: "set_override" }],
          },
        }),
      ],
    },
  ],
};
