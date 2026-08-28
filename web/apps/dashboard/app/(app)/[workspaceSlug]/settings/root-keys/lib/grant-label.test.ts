import { describe, expect, it } from "vitest";
import { grantLabel } from "./grant-label";

describe("grantLabel", () => {
  it("humanises a legacy permission string", () => {
    expect(grantLabel("api.*.create_api")).toEqual({ path: null, action: "Create API" });
  });

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
    expect(grantLabel("*")).toEqual({ path: null, action: "All permissions" });
  });
});
