import { branchDomainPattern } from "@/lib/branch-domain";
import type { CustomDomain, Domain, Environment, EnvironmentKind } from "@/lib/deploy-data";
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
