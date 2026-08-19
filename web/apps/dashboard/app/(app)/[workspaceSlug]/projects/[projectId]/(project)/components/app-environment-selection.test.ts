import { describe, expect, it } from "vitest";
import { createAppEnvironmentFilters } from "./app-environment-selection";

describe("app and environment selection", () => {
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
});
