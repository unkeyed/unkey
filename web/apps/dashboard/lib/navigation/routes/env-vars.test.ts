import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("env-var paths", () => {
  it("builds the list path", () => {
    expect(routes.envVars.list({ workspaceSlug: "acme" })).toBe("/acme/env-vars");
  });
});
