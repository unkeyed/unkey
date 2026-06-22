import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";

describe("audit-scoped paths", () => {
  it("builds the list path", () => {
    expect(routes.audit.list({ workspaceSlug: ws })).toBe("/acme/audit");
  });
});
