import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";

describe("monetization-scoped paths", () => {
  it("builds the overview path", () => {
    expect(routes.monetization.overview({ workspaceSlug: ws })).toBe("/acme/monetization");
  });
});
