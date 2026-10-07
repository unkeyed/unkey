import { describe, expect, it } from "vitest";
import { buildProjectLinks, buildWorkspaceSections } from "./leaves-projects";

const ws = "acme";
const projectId = "proj_123";

const rows = (links: ReturnType<typeof buildWorkspaceSections>) =>
  links.map(({ key, label, href, isActive }) => ({ key, label, href, isActive }));

describe("projects-first workspace sections", () => {
  it("lists projects, root keys, logs, audit log and workspace settings for an admin", () => {
    expect(rows(buildWorkspaceSections(ws, ["projects"], true))).toEqual([
      { key: "projects", label: "Projects", href: "/acme/projects", isActive: true },
      { key: "root-keys", label: "Root Keys", href: "/acme/root-keys", isActive: false },
      { key: "logs", label: "Logs", href: "/acme/logs", isActive: false },
      { key: "audit", label: "Audit Log", href: "/acme/audit", isActive: false },
      {
        key: "settings",
        label: "Workspace Settings",
        href: "/acme/settings/general",
        isActive: false,
      },
    ]);
  });

  it("omits root keys for a non-admin", () => {
    expect(rows(buildWorkspaceSections(ws, ["projects"], false))).toEqual([
      { key: "projects", label: "Projects", href: "/acme/projects", isActive: true },
      { key: "logs", label: "Logs", href: "/acme/logs", isActive: false },
      { key: "audit", label: "Audit Log", href: "/acme/audit", isActive: false },
      {
        key: "settings",
        label: "Workspace Settings",
        href: "/acme/settings/general",
        isActive: false,
      },
    ]);
  });

  it("marks root keys active on the root-keys segment", () => {
    const active = buildWorkspaceSections(ws, ["root-keys"], true).filter((link) => link.isActive);
    expect(active.map((link) => link.key)).toEqual(["root-keys"]);
  });

  it("marks logs active on the workspace logs segment", () => {
    const active = buildWorkspaceSections(ws, ["logs"], false).filter((link) => link.isActive);
    expect(active.map((link) => link.key)).toEqual(["logs"]);
  });

  it("marks nothing active outside the five sections", () => {
    expect(buildWorkspaceSections(ws, ["apis"], true).filter((link) => link.isActive)).toEqual([]);
  });
});

describe("projects-first project links", () => {
  it("lists the eight project sections in order", () => {
    expect(rows(buildProjectLinks(ws, projectId, ["projects", projectId]))).toEqual([
      { key: "apps", label: "Apps", href: "/acme/projects/proj_123", isActive: true },
      {
        key: "keyspaces",
        label: "Keyspaces",
        href: "/acme/projects/proj_123/keyspaces",
        isActive: false,
      },
      {
        key: "ratelimits",
        label: "Ratelimits",
        href: "/acme/projects/proj_123/ratelimits",
        isActive: false,
      },
      {
        key: "authorization",
        label: "Authorization",
        href: "/acme/projects/proj_123/authorization/roles",
        isActive: false,
      },
      {
        key: "identities",
        label: "Identities",
        href: "/acme/projects/proj_123/identities",
        isActive: false,
      },
      { key: "logs", label: "Logs", href: "/acme/projects/proj_123/logs", isActive: false },
      {
        key: "requests",
        label: "Requests",
        href: "/acme/projects/proj_123/requests",
        isActive: false,
      },
      {
        key: "settings",
        label: "Project Settings",
        href: "/acme/projects/proj_123/settings",
        isActive: false,
      },
    ]);
  });

  it.each([
    ["keyspaces", "keyspaces"],
    ["ratelimits", "ratelimits"],
    ["identities", "identities"],
    ["logs", "logs"],
    ["requests", "requests"],
    ["settings", "settings"],
  ])("marks %s active on its own segment", (segment, key) => {
    const active = buildProjectLinks(ws, projectId, ["projects", projectId, segment]).filter(
      (link) => link.isActive,
    );
    expect(active.map((link) => link.key)).toEqual([key]);
  });

  it("marks authorization active on both roles and permissions", () => {
    for (const leaf of ["roles", "permissions"]) {
      const active = buildProjectLinks(ws, projectId, [
        "projects",
        projectId,
        "authorization",
        leaf,
      ]).filter((link) => link.isActive);
      expect(active.map((link) => link.key)).toEqual(["authorization"]);
    }
  });
});
