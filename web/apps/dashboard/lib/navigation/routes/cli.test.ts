import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("cli paths", () => {
  it("builds the sessions path", () => {
    expect(routes.cli.sessions()).toBe("/cli/sessions");
  });
});
