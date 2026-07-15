import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";

describe("logs-scoped paths", () => {
  it("builds the list path", () => {
    expect(routes.logs.list({ workspaceSlug: ws })).toBe("/acme/logs");
  });
});
