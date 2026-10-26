import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("cli paths", () => {
  it("builds the device path", () => {
    expect(routes.cli.device()).toBe("/cli/device");
    expect(routes.cli.device({ user_code: "ABCD2345" })).toBe("/cli/device?user_code=ABCD2345");
  });
});
