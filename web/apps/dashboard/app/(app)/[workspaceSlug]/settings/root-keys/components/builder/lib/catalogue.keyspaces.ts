import { keyspaceRows } from "./catalogue.rows";
import { INSTANCE_TOKEN, type ScopeCatalogue } from "./catalogue.types";

export const keyspacesCatalogue: ScopeCatalogue = {
  scope: "keyspaces",
  label: "Keyspaces",
  allLabel: "All keyspaces",
  instanceNoun: "keyspaces",
  allInstance: "projects/*/keyspaces/*",
  groups: [{ id: "keyspaces", label: "Key management", rows: keyspaceRows(INSTANCE_TOKEN) }],
};
