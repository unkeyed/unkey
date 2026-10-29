import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("oauth paths", () => {
  it("builds the consent path", () => {
    expect(routes.oauth.consent()).toBe("/oauth/consent");
    expect(routes.oauth.consent({ user_code: "BCDFGHJKLMNPQRSTVWXZ234567" })).toBe(
      "/oauth/consent?user_code=BCDFGHJKLMNPQRSTVWXZ234567",
    );
  });
});
