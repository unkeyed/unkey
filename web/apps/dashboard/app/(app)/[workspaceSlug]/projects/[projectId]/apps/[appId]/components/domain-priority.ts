import type { CustomDomain, Domain } from "@/lib/collections";
import {
  type DisplayDomain,
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
    .sort((a, b) => a.fullyQualifiedDomainName.localeCompare(b.fullyQualifiedDomainName))
    .map(platformDisplayDomain);

  const stickyLive = platformDisplayDomains.filter(
    (d) => d.source === "platform" && d.domain.sticky === "live",
  );
  const stickyBranch = platformDisplayDomains.filter(
    (d) => d.source === "platform" && d.domain.sticky === "branch",
  );
  const restPlatform = platformDisplayDomains.filter(
    (d) => d.source === "platform" && d.domain.sticky !== "live" && d.domain.sticky !== "branch",
  );

  const all: ReadonlyArray<DisplayDomain> = [
    ...customDisplayDomains,
    ...stickyLive,
    ...stickyBranch,
    ...restPlatform,
  ];

  const primary =
    customDisplayDomains[0] ??
    platformDisplayDomains.find((d) => d.source === "platform" && d.domain.sticky === "live") ??
    platformDisplayDomains.find((d) => d.source === "platform" && d.domain.sticky === "branch") ??
    platformDisplayDomains[0] ??
    null;

  const additional = primary ? all.filter((d) => d.id !== primary.id) : [];

  return { primary, additional, all };
}
