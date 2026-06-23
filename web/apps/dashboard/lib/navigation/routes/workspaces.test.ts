import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("workspace paths", () => {
  it("builds the app root path", () => {
    expect(routes.workspaces.root()).toBe("/");
  });

  it("builds the workspace overview path", () => {
    expect(routes.workspaces.overview({ workspaceSlug: "acme" })).toBe("/acme");
  });

  it("builds the onboarding path", () => {
    expect(routes.workspaces.create()).toBe("/new");
  });
});
