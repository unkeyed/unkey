import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import {
  createAppEnvironmentUnionFilters,
  getSelectedEnvironmentIds,
  groupEnvironmentsByApp,
  toggleAppSelection,
  toggleEnvironmentSelection,
} from "./app-environment-selection";

describe("app and environment selection", () => {
  it("puts production first by kind, then sorts environment names within each app", () => {
    const appId = newId("app");
    const otherAppId = newId("app");
    const environments = [
      { appId, slug: "z-preview", kind: "preview" },
      { appId: otherAppId, slug: "preview", kind: "preview" },
      { appId, slug: "live", kind: "production" },
      { appId, slug: "a-preview", kind: "preview" },
      { appId: otherAppId, slug: "production", kind: "production" },
    ] as const;

    const grouped = groupEnvironmentsByApp(environments);

    expect(grouped.get(appId)).toEqual([environments[2], environments[3], environments[0]]);
    expect(grouped.get(otherAppId)).toEqual([environments[4], environments[1]]);
  });

  it("expresses mixed app and environment selections as an environment union", () => {
    const filters = createAppEnvironmentUnionFilters(
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

  it("expands an app to exact environment paths without dropping selections outside that app", () => {
    const projectId = newId("project");
    const appId = newId("app");
    const otherAppId = newId("app");
    const preview = `projects/${projectId}/apps/${appId}/environments/${newId("environment")}`;
    const production = `projects/${projectId}/apps/${appId}/environments/${newId("environment")}`;
    const other = `projects/${projectId}/apps/${otherAppId}/environments/${newId("environment")}`;
    const unavailable = `projects/${projectId}/apps/${otherAppId}/environments/${newId("environment")}`;
    const environments = [
      { id: preview, appId },
      { id: production, appId },
      { id: other, appId: otherAppId },
    ];
    const initial = { appIds: new Set<string>(), environmentIds: new Set([preview, unavailable]) };
    const wholeApp = toggleAppSelection(initial, appId, [preview, production]);

    expect(getSelectedEnvironmentIds(wholeApp, environments)).toEqual([
      unavailable,
      preview,
      production,
    ]);
    const partial = toggleEnvironmentSelection(wholeApp, appId, preview, [preview, production]);
    expect(getSelectedEnvironmentIds(partial, environments)).toEqual([unavailable, production]);
    expect(getSelectedEnvironmentIds(initial, environments)).toEqual([preview, unavailable]);
  });

  it("clears an app after its environment IDs have been saved and reloaded", () => {
    const appId = newId("app");
    const environmentIds = [newId("environment"), newId("environment")];
    const environments = environmentIds.map((id) => ({ id, appId }));
    const selection = { appIds: new Set<string>(), environmentIds: new Set(environmentIds) };

    expect(getSelectedEnvironmentIds(selection, environments)).toEqual(environmentIds);
    expect(
      getSelectedEnvironmentIds(toggleAppSelection(selection, appId, environmentIds), environments),
    ).toEqual([]);
    expect(
      getSelectedEnvironmentIds({ ...selection, appIds: new Set([appId]) }, environments),
    ).toEqual(environmentIds);
  });
});
