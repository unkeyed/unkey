import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("workspace paths", () => {
  it("builds the onboarding path", () => {
    expect(routes.workspaces.create()).toBe("/new");
  });
});
