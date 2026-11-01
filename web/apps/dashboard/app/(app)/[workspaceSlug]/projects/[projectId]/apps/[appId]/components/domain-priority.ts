import type { CustomDomain, Deployment, Domain } from "@/lib/deploy-data";
import {
  type DisplayDomain,
  byStickyRank,
  customDisplayDomain,
  platformDisplayDomain,
  withoutCustomDomainRoutes,
} from "./display-domain";

export type DomainPriorityContext = {
  domains: ReadonlyArray<Domain>;
  customDomains: ReadonlyArray<CustomDomain>;
  environmentId: string;
  deploymentId: string;
  currentDeploymentId: string | null;
};

export type DomainPriorityResult = {
  primary: DisplayDomain | null;
  additional: ReadonlyArray<DisplayDomain>;
  all: ReadonlyArray<DisplayDomain>;
};

export function getDomainPriority(ctx: DomainPriorityContext): DomainPriorityResult {
  const isCurrentDeployment = ctx.deploymentId === ctx.currentDeploymentId;

  const customDisplayDomains: ReadonlyArray<DisplayDomain> = isCurrentDeployment
    ? ctx.customDomains
        .filter(
          (cd) => cd.environmentId === ctx.environmentId && cd.verificationStatus === "verified",
        )
        .toSorted((a, b) => a.domain.localeCompare(b.domain))
        .map(customDisplayDomain)
    : [];

  const platformDisplayDomains: ReadonlyArray<DisplayDomain> = withoutCustomDomainRoutes(
    ctx.domains,
    ctx.customDomains,
  )
    .toSorted(
      (a, b) =>
        byStickyRank(a, b) || a.fullyQualifiedDomainName.localeCompare(b.fullyQualifiedDomainName),
    )
    .map(platformDisplayDomain);

  const all: ReadonlyArray<DisplayDomain> = [...customDisplayDomains, ...platformDisplayDomains];
  const primary = all[0] ?? null;
  const additional = primary ? all.filter((d) => d.id !== primary.id) : [];

  return { primary, additional, all };
}

// appDomains is capped and drops deployment routes first, so it only supplies
// sticky for the hostnames it still holds
export function getDeploymentDomains(
  deployment: Pick<
    Deployment,
    "id" | "projectId" | "appId" | "environmentId" | "createdAt" | "domains"
  >,
  appDomains: ReadonlyArray<Domain>,
): Domain[] {
  const byHostname = new Map(appDomains.map((d) => [d.fullyQualifiedDomainName, d]));
  return deployment.domains.map(
    (hostname) =>
      byHostname.get(hostname) ?? {
        id: hostname,
        fullyQualifiedDomainName: hostname,
        projectId: deployment.projectId,
        appId: deployment.appId,
        deploymentId: deployment.id,
        environmentId: deployment.environmentId,
        sticky: "deployment",
        createdAt: deployment.createdAt,
        updatedAt: null,
      },
  );
}
