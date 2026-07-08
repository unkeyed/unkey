import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("portal paths", () => {
  it("builds the portal root path", () => {
    expect(routes.portal.root({ workspaceSlug: "acme" })).toBe("/acme/portal");
  });
});
