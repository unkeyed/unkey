import { describe, expect, it } from "vitest";
import { grantLabel } from "./grant-label";

describe("grantLabel", () => {
  it("splits a urn into path and action", () => {
    expect(grantLabel("unkey:v1:ws_123:ratelimits/namespaces/*/overrides/*#set_override")).toEqual({
      path: "ratelimits/namespaces/*/overrides/*",
      action: "Set override",
    });
  });

  it("keeps a urn from another workspace readable", () => {
    expect(grantLabel("unkey:v1:ws_other:identities/*#read_identity")).toEqual({
      path: "identities/*",
      action: "Read identity",
    });
  });

  it("falls back to the raw grant when nothing parses", () => {
    expect(grantLabel("invalid permission data")).toEqual({
      path: null,
      action: "invalid permission data",
    });
  });
});
