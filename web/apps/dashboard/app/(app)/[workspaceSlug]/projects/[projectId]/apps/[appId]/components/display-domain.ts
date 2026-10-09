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

/**
 * A custom domain gets a frontline route once verified. That route is the
 * custom domain, never a platform alias, whatever the custom domain's status.
 */
export function withoutCustomDomainRoutes<D extends Pick<Domain, "fullyQualifiedDomainName">>(
  domains: ReadonlyArray<D>,
  customDomains: ReadonlyArray<Pick<CustomDomain, "domain">>,
): D[] {
  const custom = new Set(customDomains.map((d) => d.domain));
  return domains.filter((d) => !custom.has(d.fullyQualifiedDomainName));
}

export function listedPlatformDomains(
  domains: ReadonlyArray<Domain>,
  customDomains: ReadonlyArray<CustomDomain>,
): ListedPlatformDomain[] {
  const routes = withoutCustomDomainRoutes(domains, customDomains);
  return (["live", "environment"] as const).flatMap((sticky) =>
    routes.filter((d): d is ListedPlatformDomain => d.sticky === sticky),
  );
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
