import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("root-key paths", () => {
  it("builds the list path", () => {
    expect(routes.rootKeys.list({ workspaceSlug: "acme" })).toBe("/acme/root-keys");
  });
});
