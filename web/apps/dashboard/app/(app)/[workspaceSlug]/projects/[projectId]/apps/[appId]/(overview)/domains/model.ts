import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment, EnvironmentKind } from "@/lib/collections/deploy/environments";

export type AssignedDomain = { hostname: string; assignedTo: string };

const PLATFORM_ASSIGNMENT = {
  live: "Live deployment",
  environment: "Latest deployment",
} as const satisfies Partial<Record<Domain["sticky"], string>>;

type PlatformDomain = Domain & { sticky: keyof typeof PLATFORM_ASSIGNMENT };

function isPlatformSticky(sticky: Domain["sticky"]): sticky is PlatformDomain["sticky"] {
  return sticky in PLATFORM_ASSIGNMENT;
}

const CUSTOM_DOMAIN_ASSIGNMENT: Record<EnvironmentKind, string> = {
  production: "Live deployment",
  preview: "Latest deployment",
};

const BRANCH_DOMAIN_ASSIGNMENT = "Latest deployment of each branch";

/** A verified custom domain also gets a live route, so routes on custom hostnames are left out. */
export function platformDomainList(
  domains: ReadonlyArray<Domain>,
  customDomains: ReadonlyArray<CustomDomain>,
): PlatformDomain[] {
  const custom = new Set(customDomains.map((d) => d.domain));
  return (["live", "environment"] as const).flatMap((sticky) =>
    domains.filter(
      (d): d is PlatformDomain =>
        isPlatformSticky(d.sticky) &&
        d.sticky === sticky &&
        !custom.has(d.fullyQualifiedDomainName),
    ),
  );
}

function platformDomains(
  environment: Pick<Environment, "id">,
  domains: ReadonlyArray<Domain>,
  customDomains: ReadonlyArray<CustomDomain>,
): AssignedDomain[] {
  return platformDomainList(domains, customDomains)
    .filter((d) => d.environmentId === environment.id)
    .map((d) => ({
      hostname: d.fullyQualifiedDomainName,
      assignedTo: PLATFORM_ASSIGNMENT[d.sticky],
    }));
}

export function environmentDomains(
  environment: Pick<Environment, "id" | "kind">,
  domains: ReadonlyArray<Domain>,
  customDomains: ReadonlyArray<CustomDomain>,
): AssignedDomain[] {
  const custom = customDomains
    .filter((d) => d.environmentId === environment.id)
    .map((d) => ({ hostname: d.domain, assignedTo: CUSTOM_DOMAIN_ASSIGNMENT[environment.kind] }));
  return [...custom, ...platformDomains(environment, domains, customDomains)];
}

/**
 * The backend names the environment domain `<prefix>-<env>-<workspace>.<apex>`
 * and each branch `<prefix>-git-<branch>-<workspace>.<apex>`. Labels too long
 * for DNS are hashed instead, and those yield null.
 */
export function branchDomainPattern(
  environmentHostname: string,
  environmentSlug: string,
  workspaceSlug: string,
): string | null {
  const dot = environmentHostname.indexOf(".");
  if (dot === -1) {
    return null;
  }
  const label = environmentHostname.slice(0, dot);
  const suffix = `-${environmentSlug}-${workspaceSlug}`;
  if (!label.endsWith(suffix) || label.length === suffix.length) {
    return null;
  }
  const prefix = label.slice(0, -suffix.length);
  return `${prefix}-git-<branch>-${workspaceSlug}${environmentHostname.slice(dot)}`;
}

export function environmentPanelDomains(
  environment: Pick<Environment, "id" | "kind" | "slug">,
  domains: ReadonlyArray<Domain>,
  customDomains: ReadonlyArray<CustomDomain>,
  workspaceSlug: string,
): AssignedDomain[] {
  const listed = environmentDomains(environment, domains, customDomains);
  if (environment.kind !== "preview") {
    return listed;
  }
  const environmentDomain = domains.find(
    (d) => d.environmentId === environment.id && d.sticky === "environment",
  );
  const pattern = environmentDomain
    ? branchDomainPattern(
        environmentDomain.fullyQualifiedDomainName,
        environment.slug,
        workspaceSlug,
      )
    : null;
  return pattern
    ? [...listed, { hostname: pattern, assignedTo: BRANCH_DOMAIN_ASSIGNMENT }]
    : listed;
}
