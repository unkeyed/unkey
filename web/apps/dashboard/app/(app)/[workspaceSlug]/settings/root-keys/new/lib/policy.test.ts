import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { appsCatalogue } from "./catalogue.apps";
import { environmentsCatalogue } from "./catalogue.environments";
import { keyspacesCatalogue } from "./catalogue.keyspaces";
import { projectsCatalogue } from "./catalogue.projects";
import { ratelimitNamespacesCatalogue } from "./catalogue.ratelimit-namespaces";
import { RESOURCE_SCOPES } from "./catalogue.types";
import { workspaceCatalogue } from "./catalogue.workspace";
import {
  ALL_INSTANCES,
  actionLabel,
  countSelectedActions,
  countSelectedRows,
  environmentLabel,
  grantsPreview,
  isPolicyComplete,
  isRowActionSelected,
  newPolicy,
  policyError,
  policySummary,
  rowActions,
  selectInstances,
  setRowActions,
  setRowsActions,
  toggleRowAction,
} from "./policy";

describe("newPolicy", () => {
  it("starts on the workspace scope with all instances and no grants", () => {
    expect(newPolicy()).toEqual({
      scope: "workspace",
      instances: [ALL_INSTANCES],
      selection: {},
    });
  });
});

describe("selection helpers", () => {
  it("returns actions in the canonical order", () => {
    const selection = setRowActions({}, "role", ["delete", "read"]);
    expect(rowActions(selection, "role")).toEqual(["read", "delete"]);
  });

  it("returns nothing for an untouched row", () => {
    expect(rowActions({}, "role")).toEqual([]);
    expect(isRowActionSelected({}, "role", "read")).toBe(false);
  });

  it("drops the row once its last action is cleared", () => {
    const selection = setRowActions({}, "role", ["read"]);
    expect(toggleRowAction(selection, "role", "read", false)).toEqual({});
  });

  it("never stores an action twice", () => {
    const selection = toggleRowAction(setRowActions({}, "role", ["read"]), "role", "read", true);
    expect(rowActions(selection, "role")).toEqual(["read"]);
  });

  it("leaves other rows untouched", () => {
    const selection = setRowActions(setRowActions({}, "role", ["read"]), "identity", ["delete"]);
    expect(selection).toEqual({ role: ["read"], identity: ["delete"] });
  });

  it("applies a bulk selection to every row given", () => {
    const rows = catalogueRows(workspaceCatalogue);
    const selection = setRowsActions({}, rows, ["read", "write", "delete"]);
    expect(countSelectedActions(selection, rows)).toBe(rows.length * 3);
    expect(countSelectedRows(selection, rows)).toBe(rows.length);
  });

  it("clears a bulk selection", () => {
    const rows = catalogueRows(workspaceCatalogue);
    const selection = setRowsActions(setRowsActions({}, rows, ["read"]), rows, []);
    expect(selection).toEqual({});
    expect(countSelectedRows(selection, rows)).toBe(0);
  });
});

describe("newPolicy on an instance scope", () => {
  it("starts on all instances so it is valid at once", () => {
    expect(newPolicy("keyspaces")).toEqual({
      scope: "keyspaces",
      instances: [ALL_INSTANCES],
      selection: {},
    });
  });
});

describe("selectInstances", () => {
  it("clears named picks when all is chosen", () => {
    expect(selectInstances(["ks_1", "ks_2"], ["ks_1", "ks_2", ALL_INSTANCES])).toEqual([
      ALL_INSTANCES,
    ]);
  });

  it("clears all when a named instance is chosen", () => {
    expect(selectInstances([ALL_INSTANCES], [ALL_INSTANCES, "ks_1"])).toEqual(["ks_1"]);
  });

  it("keeps a second named pick", () => {
    expect(selectInstances(["ks_1"], ["ks_1", "ks_2"])).toEqual(["ks_1", "ks_2"]);
  });

  it("lets the last pick go", () => {
    expect(selectInstances([ALL_INSTANCES], [])).toEqual([]);
    expect(selectInstances(["ks_1"], [])).toEqual([]);
  });
});

describe("policyError", () => {
  it("asks for a permission on an empty workspace policy", () => {
    expect(policyError(newPolicy())).toBe("At least one permission required");
  });

  it("asks for an instance before a permission", () => {
    expect(policyError({ ...newPolicy("keyspaces"), instances: [] })).toBe(
      "Select one or more keyspaces",
    );
    expect(policyError({ ...newPolicy("ratelimit-namespaces"), instances: [] })).toBe(
      "Select one or more namespaces",
    );
  });

  it("is silent once a grant is ticked", () => {
    expect(
      policyError({ ...newPolicy("keyspaces"), selection: setRowActions({}, "key", ["read"]) }),
    ).toBeNull();
  });
});

describe("isPolicyComplete", () => {
  it("is false without a grant", () => {
    expect(isPolicyComplete(newPolicy())).toBe(false);
  });

  it("is true with one grant", () => {
    expect(
      isPolicyComplete({ ...newPolicy(), selection: setRowActions({}, "identity", ["read"]) }),
    ).toBe(true);
  });

  it("is false when no instance is picked", () => {
    expect(
      isPolicyComplete({
        ...newPolicy(),
        instances: [],
        selection: setRowActions({}, "identity", ["read"]),
      }),
    ).toBe(false);
  });

  it("ignores grants on rows outside the catalogue", () => {
    expect(
      isPolicyComplete({ ...newPolicy(), selection: setRowActions({}, "ghost_row", ["read"]) }),
    ).toBe(false);
  });
});

describe("actionLabel", () => {
  it("composes the ticked actions", () => {
    expect(actionLabel(["read"])).toBe("Read");
    expect(actionLabel(["read", "write"])).toBe("Read & write");
    expect(actionLabel(["read", "write", "delete"])).toBe("Read, write & delete");
    expect(actionLabel(["write", "delete"])).toBe("Write & delete");
    expect(actionLabel(["delete"])).toBe("Delete");
  });

  it("ignores the order it is given", () => {
    expect(actionLabel(["delete", "read", "write"])).toBe("Read, write & delete");
  });

  it("is empty with nothing ticked", () => {
    expect(actionLabel([])).toBe("");
  });
});

describe("policySummary", () => {
  it("names the whole workspace when every instance is covered", () => {
    expect(policySummary(newPolicy()).scopeLine).toBe("Everything in this workspace");
  });

  it("lists named instances, resolving labels when given", () => {
    const policy = { ...newPolicy(), instances: ["proj_1", "proj_2"] };
    expect(policySummary(policy).scopeLine).toBe("proj_1, proj_2");
    expect(policySummary(policy, { proj_1: "web platform" }).scopeLine).toBe(
      "web platform, proj_2",
    );
  });

  it("composes one grant per ticked row, in catalogue order", () => {
    const selection = setRowActions(
      setRowActions({}, "role", ["read", "write", "delete"]),
      "identity",
      ["read"],
    );
    expect(policySummary({ ...newPolicy(), selection }).grants).toEqual([
      "End-user identities Read",
      "Role definitions Read, write & delete",
    ]);
  });

  it("has no grants when nothing is ticked", () => {
    expect(policySummary(newPolicy()).grants).toEqual([]);
  });
});

describe("policySummary on instance scopes", () => {
  it("names all instances when nothing is narrowed", () => {
    expect(policySummary(newPolicy("keyspaces")).scopeLine).toBe("All keyspaces");
    expect(policySummary(newPolicy("ratelimit-namespaces")).scopeLine).toBe("All namespaces");
  });

  it("resolves picked instances to their labels", () => {
    const policy = { ...newPolicy("keyspaces"), instances: ["ks_1", "ks_2"] };
    expect(policySummary(policy, { ks_1: "payments" }).scopeLine).toBe("payments, ks_2");
  });

  it("composes grants from the catalogue of its own scope", () => {
    const keyspace = {
      ...newPolicy("keyspaces"),
      selection: setRowActions(setRowActions({}, "keyspace", ["read"]), "key", ["read", "write"]),
    };
    expect(policySummary(keyspace).grants).toEqual([
      "Keyspace settings Read",
      "API keys Read & write",
    ]);

    const namespace = {
      ...newPolicy("ratelimit-namespaces"),
      selection: setRowActions({}, "override", ["read", "write", "delete"]),
    };
    expect(policySummary(namespace).grants).toEqual(["Limit overrides Read, write & delete"]);
  });

  it("ignores a selection carried over from another scope", () => {
    const policy = {
      ...newPolicy("ratelimit-namespaces"),
      selection: setRowActions({}, "key", ["read"]),
    };
    expect(policySummary(policy).grants).toEqual([]);
    expect(policyError(policy)).toBe("At least one permission required");
  });
});

describe("catalogue coverage", () => {
  it("counts every row of every scope catalogue", () => {
    expect(catalogueRows(keyspacesCatalogue).map((row) => row.id)).toEqual(["keyspace", "key"]);
    expect(catalogueRows(ratelimitNamespacesCatalogue).map((row) => row.id)).toEqual([
      "namespace",
      "override",
    ]);
  });
});

describe("grantsPreview", () => {
  it("shows the first three grants and counts the rest", () => {
    expect(grantsPreview(["a", "b", "c", "d", "e"])).toEqual({
      shown: ["a", "b", "c"],
      more: 2,
    });
  });

  it("counts nothing extra when the list fits", () => {
    expect(grantsPreview(["a", "b"])).toEqual({ shown: ["a", "b"], more: 0 });
  });
});

describe("the six resource scopes", () => {
  it("lists plain nouns in container order", () => {
    expect(RESOURCE_SCOPES.map((scope) => CATALOGUES[scope].label)).toEqual([
      "Workspace",
      "Projects",
      "Apps",
      "Environments",
      "Keyspaces",
      "Ratelimit namespaces",
    ]);
  });

  it("gives every instance scope an all row and a noun for the picker", () => {
    expect(CATALOGUES.projects.allLabel).toBe("All projects");
    expect(CATALOGUES.apps.allLabel).toBe("All apps");
    expect(CATALOGUES.environments.allLabel).toBe("All environments");
    expect(CATALOGUES.projects.instanceNoun).toBe("projects");
    expect(CATALOGUES.apps.instanceNoun).toBe("apps");
    expect(CATALOGUES.environments.instanceNoun).toBe("environments");
  });

  it("counts the rows of every deploy catalogue", () => {
    expect(catalogueRows(projectsCatalogue).map((row) => row.id)).toEqual([
      "project",
      "app",
      "environment",
      "deployment",
      "domain",
      "variable",
    ]);
    expect(catalogueRows(appsCatalogue).map((row) => row.id)).toEqual([
      "app",
      "environment",
      "deployment",
      "domain",
      "variable",
    ]);
    expect(catalogueRows(environmentsCatalogue).map((row) => row.id)).toEqual([
      "environment",
      "deployment",
      "domain",
      "variable",
    ]);
  });
});

describe("environmentLabel", () => {
  it("joins the app name and the environment name with a space", () => {
    expect(environmentLabel("app_site", "preview")).toBe("app_site preview");
  });

  it("never adds a slash", () => {
    expect(environmentLabel("site", "production")).not.toContain("/");
  });

  it("falls back to the environment name alone when the app is unknown", () => {
    expect(environmentLabel(undefined, "production")).toBe("production");
  });
});

describe("policySummary on the deploy scopes", () => {
  it("names all instances of each level", () => {
    expect(policySummary(newPolicy("projects")).scopeLine).toBe("All projects");
    expect(policySummary(newPolicy("apps")).scopeLine).toBe("All apps");
    expect(policySummary(newPolicy("environments")).scopeLine).toBe("All environments");
  });

  it("comma-joins the environment labels it was given", () => {
    const policy = { ...newPolicy("environments"), instances: ["env_1", "env_2"] };
    expect(
      policySummary(policy, {
        env_1: environmentLabel("app_site", "production"),
        env_2: environmentLabel("app_api", "preview"),
      }).scopeLine,
    ).toBe("app_site production, app_api preview");
  });

  it("composes grants from the rows of its own level", () => {
    const policy = {
      ...newPolicy("apps"),
      selection: setRowActions(setRowActions({}, "app", ["read"]), "deployment", [
        "read",
        "write",
        "delete",
      ]),
    };
    expect(policySummary(policy).grants).toEqual([
      "App settings Read",
      "Deployments Read, write & delete",
    ]);
  });

  it("asks for an instance by the noun of its level", () => {
    expect(policyError({ ...newPolicy("projects"), instances: [] })).toBe(
      "Select one or more projects",
    );
    expect(policyError({ ...newPolicy("apps"), instances: [] })).toBe("Select one or more apps");
    expect(policyError({ ...newPolicy("environments"), instances: [] })).toBe(
      "Select one or more environments",
    );
  });

  it("drops a selection carried over from a neighbouring level", () => {
    const policy = {
      ...newPolicy("environments"),
      selection: setRowActions({}, "project", ["read"]),
    };
    expect(policySummary(policy).grants).toEqual([]);
    expect(isPolicyComplete(policy)).toBe(false);
  });
});
