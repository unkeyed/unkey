import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import type { Domain } from "@/lib/collections/deploy/domains";
import { describe, expect, it } from "vitest";
import { domainsPageList, domainsView } from "./model";

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

function customDomain(
  name: string,
  environmentId: string,
  verificationStatus: CustomDomain["verificationStatus"] = "verified",
): CustomDomain {
  return {
    id: name,
    domain: name,
    projectId: "proj",
    appId: "app",
    environmentId,
    verificationStatus,
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

describe("domainsPageList", () => {
  it("leaves out the routes of custom domains in any status", () => {
    const withCustomRoutes = [
      ...domains,
      domain("api.acme.dev", "env_prod", "live"),
      domain("old.acme.dev", "env_prod", "live"),
    ];
    expect(
      domainsPageList(withCustomRoutes, [
        customDomain("api.acme.dev", "env_prod"),
        customDomain("old.acme.dev", "env_prod", "failed"),
      ]).map((d) => [d.source, d.hostname]),
    ).toEqual([
      ["custom", "api.acme.dev"],
      ["custom", "old.acme.dev"],
      ["platform", "shop-acme.unkey.app"],
      ["platform", "shop-preview-acme.unkey.app"],
      ["platform", "shop-production-acme.unkey.app"],
    ]);
  });
});

describe("domainsView", () => {
  const base = {
    isLoading: false,
    failed: false,
    domains: [],
    customDomains: [],
    environments: [{ id: "env_prod", slug: "production" }],
    search: "",
    limit: 25,
  };

  it("loads before it reports a failure", () => {
    expect(domainsView({ ...base, isLoading: true, failed: true })).toEqual({ state: "loading" });
    expect(domainsView({ ...base, failed: true })).toEqual({ state: "failed" });
  });

  it("is empty without domains even while searching", () => {
    expect(domainsView({ ...base, search: "api" })).toEqual({ state: "empty" });
  });
});
