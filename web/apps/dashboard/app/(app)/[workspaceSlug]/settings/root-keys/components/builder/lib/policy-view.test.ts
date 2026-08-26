import { describe, expect, it } from "vitest";
import { isPolicyComplete, newPolicy, policyError, setRowActions } from "./policy";
import { actionLabel, environmentLabel, policySummary } from "./policy-view";

describe("actionLabel", () => {
  it("composes the ticked actions", () => {
    expect(actionLabel(["read"])).toBe("Read");
    expect(actionLabel(["read", "write"])).toBe("Read & Write");
    expect(actionLabel(["read", "write", "delete"])).toBe("Read, Write & Delete");
    expect(actionLabel(["write", "delete"])).toBe("Write & Delete");
    expect(actionLabel(["delete"])).toBe("Delete");
    expect(actionLabel(["read", "decrypt"])).toBe("Read & Decrypt");
  });

  it("ignores the order it is given", () => {
    expect(actionLabel(["delete", "read", "write"])).toBe("Read, Write & Delete");
  });

  it("is empty with nothing ticked", () => {
    expect(actionLabel([])).toBe("");
  });
});

describe("policySummary", () => {
  it("names the whole workspace when every instance is covered", () => {
    expect(policySummary(newPolicy()).scopeLine).toBe("All resources");
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
      "Role definitions Read, Write & Delete",
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
      "API keys Read & Write",
    ]);

    const namespace = {
      ...newPolicy("ratelimit-namespaces"),
      selection: setRowActions({}, "override", ["read", "write", "delete"]),
    };
    expect(policySummary(namespace).grants).toEqual(["Limit overrides Read, Write & Delete"]);
  });

  it("names the decrypt action in the summary", () => {
    const policy = {
      ...newPolicy("keyspaces"),
      selection: setRowActions({}, "key", ["read", "decrypt"]),
    };
    expect(policySummary(policy).grants).toEqual(["API keys Read & Decrypt"]);
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
      "Deployments Read, Write & Delete",
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
