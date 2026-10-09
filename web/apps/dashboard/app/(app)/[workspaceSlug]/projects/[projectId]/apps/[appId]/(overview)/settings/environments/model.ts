import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment, EnvironmentKind } from "@/lib/collections/deploy/environments";
import {
  type ListedPlatformDomain,
  listedPlatformDomains,
} from "../../../components/display-domain";

export type DomainAssignment = "live" | "latest" | "each-branch";

type AssignedDomain = { hostname: string; assignment: DomainAssignment };

const PLATFORM_ASSIGNMENT: Record<ListedPlatformDomain["sticky"], DomainAssignment> = {
  live: "live",
  environment: "latest",
};

const CUSTOM_DOMAIN_ASSIGNMENT: Record<EnvironmentKind, DomainAssignment> = {
  production: "live",
  preview: "latest",
};

export function environmentDomains(
  environment: Pick<Environment, "id" | "kind">,
  domains: ReadonlyArray<Domain>,
  customDomains: ReadonlyArray<CustomDomain>,
): AssignedDomain[] {
  const custom = customDomains
    .filter((d) => d.environmentId === environment.id)
    .map((d) => ({
      hostname: d.domain,
      assignment: CUSTOM_DOMAIN_ASSIGNMENT[environment.kind],
    }));
  const platform = listedPlatformDomains(domains, customDomains)
    .filter((d) => d.environmentId === environment.id)
    .map((d) => ({
      hostname: d.fullyQualifiedDomainName,
      assignment: PLATFORM_ASSIGNMENT[d.sticky],
    }));
  return [...custom, ...platform];
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
  return pattern ? [...listed, { hostname: pattern, assignment: "each-branch" }] : listed;
}
