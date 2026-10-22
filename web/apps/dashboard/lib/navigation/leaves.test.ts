import { describe, expect, it } from "vitest";
import {
  buildApiLinks,
  buildAppLinks,
  buildNamespaceLinks,
  buildWorkspaceSections,
} from "./leaves";
import type { ResolvedNavLink } from "./types";

const ws = "acme";
const apiId = "api_123";
const keyAuthId = "ks_456";
const namespaceId = "ns_123";
const projectId = "proj_123";

const rows = (links: ResolvedNavLink[]) =>
  links.map(({ key, label, href, isActive }) => ({ key, label, href, isActive }));

describe("workspace sections", () => {
  it("lists env vars and root keys before the audit log for an admin", () => {
    const keys = buildWorkspaceSections(ws, ["projects"], true).map((link) => link.key);
    expect(keys).toEqual([
      "projects",
      "apis",
      "ratelimits",
      "authorization",
      "logs",
      "identities",
      "env-vars",
      "root-keys",
      "audit",
      "settings",
    ]);
  });

  it("omits root keys for a non-admin", () => {
    const keys = buildWorkspaceSections(ws, ["projects"], false).map((link) => link.key);
    expect(keys).not.toContain("root-keys");
    expect(keys).toContain("env-vars");
  });

  it.each([
    ["env-vars", "/acme/env-vars"],
    ["root-keys", "/acme/root-keys"],
  ])("marks only %s active on its own segment", (segment, href) => {
    const active = buildWorkspaceSections(ws, [segment], true).filter((link) => link.isActive);
    expect(active.map((link) => [link.key, link.href])).toEqual([[segment, href]]);
  });
});

describe("keyspace links at workspace scope", () => {
  it("lists requests, keys, the portal and settings", () => {
    const links = buildApiLinks({ workspaceSlug: ws, apiId }, keyAuthId, ["apis", apiId], true);
    expect(rows(links)).toEqual([
      {
        key: "requests",
        label: "Requests",
        href: "/acme/apis/api_123",
        isActive: true,
      },
      {
        key: "keys",
        label: "Keys",
        href: "/acme/apis/api_123/keys/ks_456",
        isActive: false,
      },
      {
        key: "portal",
        label: "Customer portal",
        href: "/acme/apis/api_123/portal",
        isActive: false,
      },
      {
        key: "settings",
        label: "Settings",
        href: "/acme/apis/api_123/settings",
        isActive: false,
      },
    ]);
  });

  it("drops the portal row when portal management is off", () => {
    const links = buildApiLinks({ workspaceSlug: ws, apiId }, keyAuthId, ["apis", apiId], false);
    expect(links.map((link) => link.key)).toEqual(["requests", "keys", "settings"]);
  });

  it("disables keys until the keyspace id resolves", () => {
    const links = buildApiLinks({ workspaceSlug: ws, apiId }, undefined, ["apis", apiId], true);
    const keys = links.find((link) => link.key === "keys");
    expect(keys).toMatchObject({ href: "/acme/apis/api_123", disabled: true });
  });

  it("marks the settings row active on the settings segment", () => {
    const links = buildApiLinks(
      { workspaceSlug: ws, apiId },
      keyAuthId,
      ["apis", apiId, "settings"],
      true,
    );
    expect(links.filter((link) => link.isActive).map((link) => link.key)).toEqual(["settings"]);
  });
});

describe("keyspace links inside a project", () => {
  const segments = ["projects", projectId, "keyspaces", apiId];

  it("keeps the same rows with project-scoped hrefs", () => {
    const links = buildApiLinks({ workspaceSlug: ws, projectId, apiId }, keyAuthId, segments, true);
    expect(rows(links)).toEqual([
      {
        key: "requests",
        label: "Requests",
        href: "/acme/projects/proj_123/keyspaces/api_123",
        isActive: true,
      },
      {
        key: "keys",
        label: "Keys",
        href: "/acme/projects/proj_123/keyspaces/api_123/keys/ks_456",
        isActive: false,
      },
      {
        key: "portal",
        label: "Customer portal",
        href: "/acme/projects/proj_123/keyspaces/api_123/portal",
        isActive: false,
      },
      {
        key: "settings",
        label: "Settings",
        href: "/acme/projects/proj_123/keyspaces/api_123/settings",
        isActive: false,
      },
    ]);
  });

  it("reads the active page two segments deeper", () => {
    const links = buildApiLinks(
      { workspaceSlug: ws, projectId, apiId },
      keyAuthId,
      [...segments, "keys", keyAuthId],
      true,
    );
    expect(links.filter((link) => link.isActive).map((link) => link.key)).toEqual(["keys"]);
  });
});

describe("namespace links at workspace scope", () => {
  it("lists requests, logs, settings and overrides", () => {
    const links = buildNamespaceLinks({ workspaceSlug: ws, namespaceId }, [
      "ratelimits",
      namespaceId,
    ]);
    expect(rows(links)).toEqual([
      {
        key: "requests",
        label: "Requests",
        href: "/acme/ratelimits/ns_123",
        isActive: true,
      },
      { key: "logs", label: "Logs", href: "/acme/ratelimits/ns_123/logs", isActive: false },
      {
        key: "settings",
        label: "Settings",
        href: "/acme/ratelimits/ns_123/settings",
        isActive: false,
      },
      {
        key: "overrides",
        label: "Overrides",
        href: "/acme/ratelimits/ns_123/overrides",
        isActive: false,
      },
    ]);
  });
});

describe("namespace links inside a project", () => {
  const segments = ["projects", projectId, "ratelimits", namespaceId];

  it("keeps the same rows with project-scoped hrefs", () => {
    const links = buildNamespaceLinks({ workspaceSlug: ws, projectId, namespaceId }, segments);
    expect(rows(links)).toEqual([
      {
        key: "requests",
        label: "Requests",
        href: "/acme/projects/proj_123/ratelimits/ns_123",
        isActive: true,
      },
      {
        key: "logs",
        label: "Logs",
        href: "/acme/projects/proj_123/ratelimits/ns_123/logs",
        isActive: false,
      },
      {
        key: "settings",
        label: "Settings",
        href: "/acme/projects/proj_123/ratelimits/ns_123/settings",
        isActive: false,
      },
      {
        key: "overrides",
        label: "Overrides",
        href: "/acme/projects/proj_123/ratelimits/ns_123/overrides",
        isActive: false,
      },
    ]);
  });

  it("reads the active page two segments deeper", () => {
    const links = buildNamespaceLinks({ workspaceSlug: ws, projectId, namespaceId }, [
      ...segments,
      "overrides",
    ]);
    expect(links.filter((link) => link.isActive).map((link) => link.key)).toEqual(["overrides"]);
  });
});

describe("buildAppLinks", () => {
  const appId = "app_456";

  it("keeps App settings active on its sub-pages", () => {
    const links = buildAppLinks(
      ws,
      projectId,
      appId,
      ["projects", projectId, "apps", appId, "settings", "environments"],
      false,
    );
    expect(links.filter((link) => link.isActive).map((link) => link.key)).toEqual(["settings"]);
  });

  it("shows Metrics only when the app metrics flag is on", () => {
    const segments = ["projects", projectId, "apps", appId, "metrics"];
    const keys = (enabled: boolean) =>
      buildAppLinks(ws, projectId, appId, segments, enabled).map((link) => link.key);
    expect(keys(false)).not.toContain("metrics");
    expect(keys(true)).toContain("metrics");
    expect(
      buildAppLinks(ws, projectId, appId, segments, true).find((link) => link.isActive)?.key,
    ).toBe("metrics");
  });
});
