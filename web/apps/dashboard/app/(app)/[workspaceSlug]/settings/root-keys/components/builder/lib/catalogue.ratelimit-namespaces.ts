import { namespaceRows } from "./catalogue.rows";
import { INSTANCE_TOKEN, type ScopeCatalogue } from "./catalogue.types";

export const ratelimitNamespacesCatalogue: ScopeCatalogue = {
  scope: "ratelimit-namespaces",
  label: "Ratelimit namespaces",
  allLabel: "All namespaces",
  instanceNoun: "namespaces",
  allInstance: "projects/*/ratelimits/namespaces/*",
  groups: [{ id: "ratelimits", label: "Rate limiting", rows: namespaceRows(INSTANCE_TOKEN) }],
};
