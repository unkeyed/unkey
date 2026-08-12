import { describe, expect, it, vi } from "vitest";
import { redirectToSignIn, sanitizeRedirectPath } from "../redirect-utils";

describe("sanitizeRedirectPath", () => {
  it.each([
    "/apis",
    "/workspace/settings/team",
    "/workspace/settings/team?tab=pending",
    "/workspace/~member",
  ])("preserves a canonical internal path: %s", (path) => {
    expect(sanitizeRedirectPath(path)).toBe(path);
  });

  it.each([
    "https://example.com/path",
    "//example.com/path",
    "/\\example.com",
    "/%5cexample.com",
    "/workspace/../admin",
    "/workspace/%2e%2e/admin",
    "/workspace/%2fadmin",
    "/workspace/%252fadmin",
    "/workspace/%00admin",
    "/workspace/\u2028admin",
    "/workspace?tab=one&tab=two",
    "/workspace?bad=%",
  ])("rejects an unsafe return path: %s", (path) => {
    expect(sanitizeRedirectPath(path)).toBe("/apis");
  });
});

describe("redirectToSignIn", () => {
  it("uses a document navigation and preserves the current path", () => {
    const assign = vi.fn();

    redirectToSignIn({
      pathname: "/workspace/apis",
      search: "?tab=active",
      assign,
    });

    expect(assign).toHaveBeenCalledWith(
      "/auth/sign-in?redirect=%2Fworkspace%2Fapis%3Ftab%3Dactive",
    );
  });
});
