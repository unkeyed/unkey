import type { CustomDomain, Domain } from "@/lib/collections";
import { describe, expect, test } from "vitest";
import { getDomainPriority } from "./domain-priority";

function makeDomain(overrides: Partial<Domain> = {}): Domain {
  return {
    id: "dom-1",
    fullyQualifiedDomainName: "a.example.com",
    projectId: "proj-1",
    deploymentId: "dep-1",
    environmentId: "env-1",
    sticky: "none",
    createdAt: 0,
    updatedAt: null,
    ...overrides,
  };
}

function makeCustomDomain(overrides: Partial<CustomDomain> = {}): CustomDomain {
  return {
    id: "cd-1",
    domain: "custom.example.com",
    workspaceId: "ws-1",
    projectId: "proj-1",
    appId: "app-1",
    environmentId: "env-1",
    verificationStatus: "verified",
    verificationToken: "tok",
    ownershipVerified: true,
    cnameVerified: true,
    targetCname: "target.example.com",
    checkAttempts: 0,
    lastCheckedAt: null,
    verificationError: null,
    domainConnectProvider: null,
    domainConnectUrl: null,
    createdAt: 0,
    updatedAt: null,
    ...overrides,
  };
}

const baseCtx = {
  domains: [] as Domain[],
  customDomains: [] as CustomDomain[],
  environmentId: "env-1",
};

describe("getDomainPriority", () => {
  const cases = [
    {
      name: "no domains → primary is null",
      ctx: baseCtx,
      expected: { primary: null, additionalCount: 0, allCount: 0 },
    },
    {
      name: "only platform domains, no sticky → picks first alphabetically",
      ctx: {
        ...baseCtx,
        domains: [
          makeDomain({ id: "d-b", fullyQualifiedDomainName: "b.example.com" }),
          makeDomain({ id: "d-a", fullyQualifiedDomainName: "a.example.com" }),
        ],
      },
      expected: { primaryId: "d-a", additionalCount: 1, allCount: 2 },
    },
    {
      name: "platform with sticky=live → picks live",
      ctx: {
        ...baseCtx,
        domains: [
          makeDomain({ id: "d-a", fullyQualifiedDomainName: "a.example.com" }),
          makeDomain({
            id: "d-live",
            fullyQualifiedDomainName: "live.example.com",
            sticky: "live",
          }),
        ],
      },
      expected: { primaryId: "d-live", additionalCount: 1, allCount: 2 },
    },
    {
      name: "platform with sticky=branch → picks branch when no live",
      ctx: {
        ...baseCtx,
        domains: [
          makeDomain({ id: "d-a", fullyQualifiedDomainName: "a.example.com" }),
          makeDomain({
            id: "d-branch",
            fullyQualifiedDomainName: "branch.example.com",
            sticky: "branch",
          }),
        ],
      },
      expected: { primaryId: "d-branch", additionalCount: 1, allCount: 2 },
    },
    {
      name: "verified custom domain → picks custom over live",
      ctx: {
        ...baseCtx,
        domains: [
          makeDomain({
            id: "d-live",
            fullyQualifiedDomainName: "live.example.com",
            sticky: "live",
          }),
        ],
        customDomains: [makeCustomDomain({ id: "cd-1", domain: "custom.example.com" })],
      },
      expected: { primaryId: "cd-1", additionalCount: 1, allCount: 2 },
    },
    {
      name: "unverified custom domain → skipped, falls to platform",
      ctx: {
        ...baseCtx,
        domains: [makeDomain({ id: "d-a", fullyQualifiedDomainName: "a.example.com" })],
        customDomains: [
          makeCustomDomain({
            id: "cd-1",
            domain: "custom.example.com",
            verificationStatus: "pending",
          }),
        ],
      },
      expected: { primaryId: "d-a", additionalCount: 0, allCount: 1 },
    },
    {
      name: "custom domain wrong environmentId → skipped",
      ctx: {
        ...baseCtx,
        domains: [makeDomain({ id: "d-a", fullyQualifiedDomainName: "a.example.com" })],
        customDomains: [
          makeCustomDomain({
            id: "cd-1",
            domain: "custom.example.com",
            environmentId: "env-other",
          }),
        ],
      },
      expected: { primaryId: "d-a", additionalCount: 0, allCount: 1 },
    },
    {
      name: "mix of everything → verified custom wins, all ordered by priority",
      ctx: {
        ...baseCtx,
        domains: [
          makeDomain({ id: "d-a", fullyQualifiedDomainName: "a.example.com" }),
          makeDomain({
            id: "d-live",
            fullyQualifiedDomainName: "live.example.com",
            sticky: "live",
          }),
          makeDomain({
            id: "d-branch",
            fullyQualifiedDomainName: "branch.example.com",
            sticky: "branch",
          }),
        ],
        customDomains: [
          makeCustomDomain({ id: "cd-1", domain: "custom.example.com" }),
          makeCustomDomain({
            id: "cd-bad",
            domain: "bad.example.com",
            verificationStatus: "failed",
          }),
          makeCustomDomain({
            id: "cd-wrong-env",
            domain: "wrong.example.com",
            environmentId: "env-other",
          }),
        ],
      },
      expected: {
        primaryId: "cd-1",
        additionalCount: 3,
        allCount: 4,
        allIds: ["cd-1", "d-live", "d-branch", "d-a"],
      },
    },
  ] as const;

  for (const { name, ctx, expected } of cases) {
    test(name, () => {
      const result = getDomainPriority(ctx);

      if ("primary" in expected && expected.primary === null) {
        expect(result.primary).toBeNull();
      } else if ("primaryId" in expected) {
        expect(result.primary?.id).toBe(expected.primaryId);
      }

      expect(result.additional).toHaveLength(expected.additionalCount);
      expect(result.all).toHaveLength(expected.allCount);

      if ("allIds" in expected) {
        expect(result.all.map((d) => d.id)).toEqual(expected.allIds);
      }
    });
  }

  test("verified custom domain deduplicates matching platform domain", () => {
    const result = getDomainPriority({
      ...baseCtx,
      domains: [
        makeDomain({
          id: "d-custom-route",
          fullyQualifiedDomainName: "custom.example.com",
          sticky: "live",
        }),
        makeDomain({ id: "d-a", fullyQualifiedDomainName: "a.example.com" }),
      ],
      customDomains: [makeCustomDomain({ id: "cd-1", domain: "custom.example.com" })],
    });

    expect(result.all).toHaveLength(2);
    expect(result.all.map((d) => d.id)).toEqual(["cd-1", "d-a"]);
    expect(result.primary?.id).toBe("cd-1");
  });

  test("all order: custom → sticky live → sticky branch → rest alphabetically", () => {
    const result = getDomainPriority({
      ...baseCtx,
      domains: [
        makeDomain({ id: "d-z", fullyQualifiedDomainName: "z.example.com" }),
        makeDomain({ id: "d-a", fullyQualifiedDomainName: "a.example.com" }),
        makeDomain({
          id: "d-live-2",
          fullyQualifiedDomainName: "live2.example.com",
          sticky: "live",
        }),
        makeDomain({
          id: "d-live-1",
          fullyQualifiedDomainName: "live1.example.com",
          sticky: "live",
        }),
        makeDomain({
          id: "d-branch",
          fullyQualifiedDomainName: "branch.example.com",
          sticky: "branch",
        }),
      ],
      customDomains: [
        makeCustomDomain({ id: "cd-b", domain: "b-custom.example.com" }),
        makeCustomDomain({ id: "cd-a", domain: "a-custom.example.com" }),
      ],
    });

    expect(result.all.map((d) => d.id)).toEqual([
      "cd-a",
      "cd-b",
      "d-live-1",
      "d-live-2",
      "d-branch",
      "d-a",
      "d-z",
    ]);
  });
});
