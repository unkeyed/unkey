import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("cli paths", () => {
  it("builds the device and sessions paths", () => {
    expect(routes.cli.device()).toBe("/cli/device");
    expect(routes.cli.device({ user_code: "ABCD2345" })).toBe("/cli/device?user_code=ABCD2345");
    expect(routes.cli.sessions()).toBe("/cli/sessions");
  });
});
