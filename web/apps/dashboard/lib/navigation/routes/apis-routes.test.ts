import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";
const apiId = "api_123";
const keyAuthId = "ks_456";
const keyId = "key_789";

describe("api-scoped paths", () => {
  it("builds the list and api base paths", () => {
    expect(routes.apis.list({ workspaceSlug: ws })).toBe("/acme/apis");
    expect(routes.apis.detail({ workspaceSlug: ws, apiId })).toBe("/acme/apis/api_123");
  });

  it("appends the new query when flagged", () => {
    expect(routes.apis.list({ workspaceSlug: ws, new: true })).toBe("/acme/apis?new=true");
  });

  it("builds the settings path", () => {
    expect(routes.apis.settings({ workspaceSlug: ws, apiId })).toBe("/acme/apis/api_123/settings");
  });
});

describe("key-scoped paths", () => {
  it("scopes to a keyspace", () => {
    expect(routes.apis.keys.list({ workspaceSlug: ws, apiId, keyAuthId })).toBe(
      "/acme/apis/api_123/keys/ks_456",
    );
  });

  it("scopes to a single key", () => {
    expect(routes.apis.keys.detail({ workspaceSlug: ws, apiId, keyAuthId, keyId })).toBe(
      "/acme/apis/api_123/keys/ks_456/key_789",
    );
  });
});
