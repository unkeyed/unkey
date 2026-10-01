import { identityRows } from "./catalogue.rows";
import type { ScopeCatalogue } from "./catalogue.types";

// Identities hang off a project, and the dashboard has no project-by-project
// identity picker, so this scope covers every project at once.
export const identitiesCatalogue: ScopeCatalogue = {
  scope: "identities",
  label: "Identities",
  allLabel: "All identities",
  instanceNoun: null,
  allInstance: "*",
  groups: [{ id: "identities", label: "Identities", rows: identityRows("projects/*") }],
};
