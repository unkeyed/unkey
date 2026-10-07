import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import type { Domain } from "@/lib/collections/deploy/domains";
import { describe, expect, it } from "vitest";
import { branchDomainPattern, environmentDomains, environmentPanelDomains } from "./model";

const production = { id: "env_prod", kind: "production" as const };
const preview = { id: "env_prev", kind: "preview" as const };

function domain(fqdn: string, environmentId: string, sticky: Domain["sticky"]): Domain {
  return {
    id: fqdn,
    fullyQualifiedDomainName: fqdn,
    projectId: "proj",
    appId: "app",
    deploymentId: "dep",
    environmentId,
    sticky,
    createdAt: 0,
    updatedAt: null,
  };
}

function customDomain(name: string, environmentId: string): CustomDomain {
  return {
    id: name,
    domain: name,
    projectId: "proj",
    appId: "app",
    environmentId,
    verificationStatus: "verified",
    dnsRecords: [],
    verificationError: null,
    domainConnectProvider: null,
    domainConnectUrl: null,
    createdAt: 0,
    updatedAt: null,
  };
}

const domains = [
  domain("shop-preview-acme.unkey.app", "env_prev", "environment"),
  domain("shop-production-acme.unkey.app", "env_prod", "environment"),
  domain("shop-git-main-acme.unkey.app", "env_prod", "branch"),
  domain("shop-git-abc1234-acme.unkey.app", "env_prod", "none"),
  domain("shop-acme.unkey.app", "env_prod", "live"),
];

describe("environmentDomains", () => {
  it("lists live then environment domains of one environment", () => {
    expect(environmentDomains(production, domains, [])).toEqual([
      { hostname: "shop-acme.unkey.app", assignedTo: "Live deployment" },
      { hostname: "shop-production-acme.unkey.app", assignedTo: "Latest deployment" },
    ]);
  });

  it("puts the environment's custom domains first", () => {
    const custom = [
      customDomain("api.acme.dev", "env_prod"),
      customDomain("pr.acme.dev", "env_prev"),
    ];
    expect(environmentDomains(production, domains, custom)).toEqual([
      { hostname: "api.acme.dev", assignedTo: "Live deployment" },
      { hostname: "shop-acme.unkey.app", assignedTo: "Live deployment" },
      { hostname: "shop-production-acme.unkey.app", assignedTo: "Latest deployment" },
    ]);
    expect(environmentDomains(preview, domains, custom)).toEqual([
      { hostname: "pr.acme.dev", assignedTo: "Latest deployment" },
      { hostname: "shop-preview-acme.unkey.app", assignedTo: "Latest deployment" },
    ]);
  });

  it("is empty for an environment without domains", () => {
    expect(environmentDomains({ id: "env_x", kind: "preview" }, domains, [])).toEqual([]);
  });
});

describe("environmentPanelDomains", () => {
  it("adds the per-branch pattern for preview", () => {
    expect(environmentPanelDomains({ ...preview, slug: "preview" }, domains, [], "acme")).toEqual([
      { hostname: "shop-preview-acme.unkey.app", assignedTo: "Latest deployment" },
      {
        hostname: "shop-git-<branch>-acme.unkey.app",
        assignedTo: "Latest deployment of each branch",
      },
    ]);
  });

  it("lists production domains without a branch pattern", () => {
    expect(
      environmentPanelDomains({ ...production, slug: "production" }, domains, [], "acme"),
    ).toEqual([
      { hostname: "shop-acme.unkey.app", assignedTo: "Live deployment" },
      { hostname: "shop-production-acme.unkey.app", assignedTo: "Latest deployment" },
    ]);
  });

  it("omits the pattern when the environment domain does not match", () => {
    expect(environmentPanelDomains({ ...preview, slug: "preview" }, domains, [], "other")).toEqual([
      { hostname: "shop-preview-acme.unkey.app", assignedTo: "Latest deployment" },
    ]);
  });
});

describe("branchDomainPattern", () => {
  it("swaps the environment slug for the branch placeholder", () => {
    expect(branchDomainPattern("shop-preview-acme.unkey.app", "preview", "acme")).toBe(
      "shop-git-<branch>-acme.unkey.app",
    );
  });

  it("only replaces the slug next to the workspace", () => {
    expect(branchDomainPattern("my-preview-shop-preview-acme.unkey.app", "preview", "acme")).toBe(
      "my-preview-shop-git-<branch>-acme.unkey.app",
    );
  });

  it("returns null for hashed or unexpected hostnames", () => {
    expect(branchDomainPattern("shop-a1b2c3d4-acme.unkey.app", "preview", "acme")).toBeNull();
    expect(branchDomainPattern("preview-acme.unkey.app", "preview", "acme")).toBeNull();
    expect(branchDomainPattern("localhost", "preview", "acme")).toBeNull();
  });
});
