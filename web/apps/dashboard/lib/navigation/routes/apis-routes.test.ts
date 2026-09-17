import { describe, expect, it } from "vitest";
import { routes } from "./index";

const ws = "acme";
const apiId = "api_123";
const keyAuthId = "ks_456";
const keyId = "key_789";
const projectId = "proj_123";

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

  it("builds the portal path", () => {
    expect(routes.apis.portal({ workspaceSlug: ws, apiId })).toBe("/acme/apis/api_123/portal");
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

describe("project-scoped api paths", () => {
  it("builds the list and api base paths inside the project", () => {
    expect(routes.apis.list({ workspaceSlug: ws, projectId })).toBe(
      "/acme/projects/proj_123/keyspaces",
    );
    expect(routes.apis.detail({ workspaceSlug: ws, projectId, apiId })).toBe(
      "/acme/projects/proj_123/keyspaces/api_123",
    );
  });

  it("appends the new query inside the project", () => {
    expect(routes.apis.list({ workspaceSlug: ws, projectId, new: true })).toBe(
      "/acme/projects/proj_123/keyspaces?new=true",
    );
  });

  it("builds the settings and portal paths inside the project", () => {
    expect(routes.apis.settings({ workspaceSlug: ws, projectId, apiId })).toBe(
      "/acme/projects/proj_123/keyspaces/api_123/settings",
    );
    expect(routes.apis.portal({ workspaceSlug: ws, projectId, apiId })).toBe(
      "/acme/projects/proj_123/keyspaces/api_123/portal",
    );
  });

  it("builds key paths inside the project", () => {
    expect(routes.apis.keys.list({ workspaceSlug: ws, projectId, apiId, keyAuthId })).toBe(
      "/acme/projects/proj_123/keyspaces/api_123/keys/ks_456",
    );
    expect(routes.apis.keys.detail({ workspaceSlug: ws, projectId, apiId, keyAuthId, keyId })).toBe(
      "/acme/projects/proj_123/keyspaces/api_123/keys/ks_456/key_789",
    );
  });

  it("keeps workspace paths when projectId is absent", () => {
    expect(routes.apis.detail({ workspaceSlug: ws, projectId: undefined, apiId })).toBe(
      "/acme/apis/api_123",
    );
  });
});
