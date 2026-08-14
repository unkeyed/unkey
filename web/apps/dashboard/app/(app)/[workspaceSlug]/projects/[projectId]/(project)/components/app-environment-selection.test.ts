import { describe, expect, it } from "vitest";
import { createAppEnvironmentSearchItems } from "./app-environment-search-items";
import { createAppEnvironmentFilters, groupEnvironmentsByApp } from "./app-environment-selection";

describe("app and environment selection", () => {
  it("groups environments by app and sorts them by slug", () => {
    const grouped = groupEnvironmentsByApp([
      { id: "env_2", appId: "app_1", slug: "production" },
      { id: "env_1", appId: "app_1", slug: "development" },
      { id: "env_3", appId: "app_2", slug: "staging" },
    ]);

    expect(grouped.get("app_1")?.map((environment) => environment.id)).toEqual(["env_1", "env_2"]);
    expect(grouped.get("app_2")?.map((environment) => environment.id)).toEqual(["env_3"]);
  });

  it("expresses mixed app and environment selections as an environment union", () => {
    const filters = createAppEnvironmentFilters(
      {
        appIds: new Set(["app_1"]),
        environmentIds: new Set(["env_3"]),
      },
      new Map([
        ["app_1", ["env_1", "env_2"]],
        ["app_2", ["env_3", "env_4"]],
      ]),
      (field, value) => ({ field, value }),
    );

    expect(filters).toEqual([
      { field: "environmentId", value: "env_3" },
      { field: "environmentId", value: "env_1" },
      { field: "environmentId", value: "env_2" },
    ]);
  });

  it("builds shared searchable app and environment options", () => {
    const selections: Array<{ appIds: Set<string>; environmentIds: Set<string> }> = [];
    const items = createAppEnvironmentSearchItems({
      apps: [{ appId: "app_1", name: "API" }],
      environmentsByAppId: new Map([
        [
          "app_1",
          [
            { id: "env_1", slug: "development" },
            { id: "env_2", slug: "production" },
          ],
        ],
      ]),
      selection: {
        appIds: new Set(),
        environmentIds: new Set(["env_1"]),
      },
      onSelectionChange: (selection) => selections.push(selection),
    });

    expect(items.map(({ id, path }) => ({ id, path }))).toEqual([
      { id: "app:app_1", path: ["App"] },
      { id: "environment:env_1", path: ["App", "API"] },
      { id: "environment:env_2", path: ["App", "API"] },
    ]);
    expect(items.map((item) => (item.kind === "option" ? item.checked : false))).toEqual([
      false,
      true,
      false,
    ]);

    const appItem = items[0];
    if (appItem?.kind === "option") {
      appItem.onSelect();
    }
    expect(selections[0]?.appIds).toEqual(new Set(["app_1"]));
    expect(selections[0]?.environmentIds).toEqual(new Set());
  });
});
