import { describe, expect, it } from "vitest";
import { CATALOGUES, catalogueRows } from "./catalogue";
import { appsCatalogue, environmentsCatalogue, projectsCatalogue } from "./catalogue.deploy";
import { keyspacesCatalogue } from "./catalogue.keyspaces";
import { ratelimitNamespacesCatalogue } from "./catalogue.ratelimit-namespaces";
import {
  ACTIONS,
  ALL_INSTANCES,
  CRUD_ACTIONS,
  RESOURCE_SCOPES,
  rowOffers,
} from "./catalogue.types";
import { workspaceCatalogue } from "./catalogue.workspace";
import {
  countSelectedActions,
  isPolicyComplete,
  newPolicy,
  policyError,
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

  it("applies a bulk selection to every row that offers it", () => {
    const rows = catalogueRows(workspaceCatalogue);
    const selection = setRowsActions({}, rows, CRUD_ACTIONS);
    const offered = rows.reduce(
      (total, row) => total + CRUD_ACTIONS.filter((action) => rowOffers(row, action)).length,
      0,
    );
    expect(countSelectedActions(selection, rows)).toBe(offered);
  });

  it("never bulk-selects an action a row does not offer", () => {
    const selection = setRowsActions({}, catalogueRows(workspaceCatalogue), ACTIONS);
    expect(selection.key).toEqual(["read", "write", "delete", "verify", "decrypt"]);
    expect(selection.ratelimit_namespace).toEqual(["read", "write", "delete", "limit"]);
    expect(selection.keyspace).toEqual(["read", "write", "delete"]);
    expect(selection.keyspace_log).toEqual(["read"]);
    expect(selection.identity).toEqual(["read", "write", "delete"]);
  });

  it("clears a bulk selection", () => {
    const rows = catalogueRows(workspaceCatalogue);
    const selection = setRowsActions(setRowsActions({}, rows, ["read"]), rows, []);
    expect(selection).toEqual({});
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

describe("catalogue coverage", () => {
  it("counts every row of every scope catalogue", () => {
    expect(catalogueRows(keyspacesCatalogue).map((row) => row.id)).toEqual([
      "keyspace",
      "keyspace_log",
      "key",
    ]);
    expect(catalogueRows(ratelimitNamespacesCatalogue).map((row) => row.id)).toEqual([
      "ratelimit_namespace",
      "ratelimit_log",
      "ratelimit_override",
    ]);
  });
});

describe("the eight resource scopes", () => {
  it("lists plain nouns in container order", () => {
    expect(RESOURCE_SCOPES.map((scope) => CATALOGUES[scope].label)).toEqual([
      "Workspace",
      "Projects",
      "Apps",
      "Environments",
      "Keyspaces",
      "Ratelimit namespaces",
      "Identities",
      "RBAC",
    ]);
  });

  it("leaves the container-less scopes without a picker", () => {
    expect(CATALOGUES.workspace.instanceNoun).toBeNull();
    expect(CATALOGUES.identities.instanceNoun).toBeNull();
    expect(CATALOGUES.rbac.instanceNoun).toBeNull();
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
      "variable",
      "domain",
      "deployment",
      "deployment_log",
      "gateway_log",
      "gateway_policy",
      "keyspace",
      "keyspace_log",
      "key",
      "ratelimit_namespace",
      "ratelimit_log",
      "ratelimit_override",
      "identity",
      "role",
      "permission",
    ]);
    expect(catalogueRows(appsCatalogue).map((row) => row.id)).toEqual([
      "app",
      "environment",
      "variable",
      "domain",
      "deployment",
      "deployment_log",
      "gateway_log",
      "gateway_policy",
    ]);
    expect(catalogueRows(environmentsCatalogue).map((row) => row.id)).toEqual([
      "environment",
      "variable",
      "domain",
      "deployment",
      "deployment_log",
      "gateway_log",
      "gateway_policy",
    ]);
  });
});
