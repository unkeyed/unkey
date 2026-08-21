import { INSTANCE_TOKEN, type ScopeCatalogue } from "./catalogue.types";

const NAMESPACE_PATH = `ratelimits/namespaces/${INSTANCE_TOKEN}`;

export const ratelimitNamespacesCatalogue: ScopeCatalogue = {
  scope: "ratelimit-namespaces",
  label: "Ratelimit namespaces",
  allLabel: "All namespaces",
  instanceNoun: "namespaces",
  groups: [
    {
      id: "namespace",
      label: "Namespace",
      rows: [
        {
          id: "namespace",
          label: "Namespace",
          description: "Namespace settings. Taking a ratelimit counts as read.",
          path: NAMESPACE_PATH,
          resource: "namespace",
          actions: {
            read: [{ name: "read_namespace" }, { name: "limit" }],
            write: [{ name: "update_namespace" }],
            delete: [{ name: "delete_namespace" }],
          },
        },
        {
          id: "override",
          label: "Overrides",
          description: "Per-identifier limit overrides.",
          path: `${NAMESPACE_PATH}/overrides/*`,
          resource: "override",
          actions: {
            read: [{ name: "read_override" }],
            write: [{ name: "set_override" }],
            delete: [{ name: "delete_override" }],
          },
        },
      ],
    },
  ],
};
