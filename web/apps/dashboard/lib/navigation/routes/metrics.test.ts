import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("metrics-scoped paths", () => {
  it("builds the list path", () => {
    expect(routes.metrics.list({ workspaceSlug: "acme" })).toBe("/acme/metrics");
  });
});
