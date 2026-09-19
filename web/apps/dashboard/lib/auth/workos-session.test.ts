import { describe, expect, it } from "vitest";
import { AUTHKIT_MIDDLEWARE_HEADER, hasAuthkitMiddleware } from "./workos-session";

describe("hasAuthkitMiddleware", () => {
  it("is true when the AuthKit middleware stamped the request", () => {
    expect(hasAuthkitMiddleware(new Headers({ [AUTHKIT_MIDDLEWARE_HEADER]: "true" }))).toBe(true);
  });

  it("is false for requests the middleware matcher skipped", () => {
    expect(hasAuthkitMiddleware(new Headers())).toBe(false);
  });
});
