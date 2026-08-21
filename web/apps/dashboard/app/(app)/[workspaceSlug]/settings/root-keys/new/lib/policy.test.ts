import { describe, expect, it } from "vitest";
import { catalogueRows } from "./catalogue";
import { workspaceCatalogue } from "./catalogue.workspace";
import {
  ALL_INSTANCES,
  actionLabel,
  countSelectedActions,
  countSelectedRows,
  grantsPreview,
  isPolicyComplete,
  isRowActionSelected,
  newPolicy,
  policySummary,
  rowActions,
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
      "Identities Read",
      "Roles Read, write & delete",
    ]);
  });

  it("has no grants when nothing is ticked", () => {
    expect(policySummary(newPolicy()).grants).toEqual([]);
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
