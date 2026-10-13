import type { CustomDomain, Domain } from "@/lib/collections";

export type DisplayDomainCustom = {
  source: "custom";
  id: string;
  hostname: string;
  url: string;
  environmentId: string;
  customDomain: CustomDomain;
};

type DisplayDomainPlatform = {
  source: "platform";
  id: string;
  hostname: string;
  url: string;
  environmentId: string;
  domain: Domain;
};

export type DisplayDomain = DisplayDomainCustom | DisplayDomainPlatform;

export type ListedPlatformDomain = Domain & { sticky: "live" | "environment" };

function domainUrl(hostname: string): string {
  return `https://${hostname}`;
}

export function withoutCustomDomainRoutes<D extends Pick<Domain, "fullyQualifiedDomainName">>(
  domains: ReadonlyArray<D>,
  customDomains: ReadonlyArray<Pick<CustomDomain, "domain">>,
): D[] {
  const custom = new Set(customDomains.map((d) => d.domain));
  return domains.filter((d) => !custom.has(d.fullyQualifiedDomainName));
}

const STICKY_RANK: Record<Domain["sticky"], number> = {
  live: 0,
  branch: 1,
  environment: 2,
  deployment: 2,
  none: 2,
};

export function byStickyRank(a: Pick<Domain, "sticky">, b: Pick<Domain, "sticky">): number {
  return STICKY_RANK[a.sticky] - STICKY_RANK[b.sticky];
}

export function listedPlatformDomains(
  domains: ReadonlyArray<Domain>,
  customDomains: ReadonlyArray<CustomDomain>,
): ListedPlatformDomain[] {
  return withoutCustomDomainRoutes(domains, customDomains)
    .filter((d): d is ListedPlatformDomain => d.sticky === "live" || d.sticky === "environment")
    .toSorted(byStickyRank);
}

export function customDisplayDomain(customDomain: CustomDomain): DisplayDomainCustom {
  return {
    source: "custom",
    id: customDomain.id,
    hostname: customDomain.domain,
    url: domainUrl(customDomain.domain),
    environmentId: customDomain.environmentId,
    customDomain,
  };
}

export function platformDisplayDomain(domain: Domain): DisplayDomainPlatform {
  return {
    source: "platform",
    id: domain.id,
    hostname: domain.fullyQualifiedDomainName,
    url: domainUrl(domain.fullyQualifiedDomainName),
    environmentId: domain.environmentId,
    domain,
  };
}
