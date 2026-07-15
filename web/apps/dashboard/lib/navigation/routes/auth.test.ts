import { describe, expect, it } from "vitest";
import { routes } from "./index";

describe("auth paths", () => {
  it("drops the optional catch-all to yield the base sign-in path", () => {
    expect(routes.auth.signIn()).toBe("/auth/sign-in");
  });
});
