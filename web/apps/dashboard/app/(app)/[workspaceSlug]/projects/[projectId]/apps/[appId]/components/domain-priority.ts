import type { CustomDomain, Domain } from "@/lib/collections";

type DisplayDomainCustom = {
  source: "custom";
  id: string;
  hostname: string;
  url: string;
  customDomain: CustomDomain;
};

type DisplayDomainPlatform = {
  source: "platform";
  id: string;
  hostname: string;
  url: string;
  domain: Domain;
};

export type DisplayDomain = DisplayDomainCustom | DisplayDomainPlatform;

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

/**
 * Returns the app-level aliases shown on the overview together with immutable
 * URLs for the displayed deployment. The `deployment` sticky value identifies
 * an immutable URL, so routes of that type from older deployments are excluded.
 */
export function getAppOverviewDomains(
  domains: ReadonlyArray<Domain>,
  deploymentId: string,
): ReadonlyArray<Domain> {
  return domains.filter(
    (domain) =>
      domain.deploymentId === deploymentId ||
      (domain.sticky !== "none" && domain.sticky !== "deployment"),
  );
}

export function getDomainPriority(ctx: DomainPriorityContext): DomainPriorityResult {
  const isCurrentDeployment = ctx.deploymentId === ctx.currentDeploymentId;

  const customDisplayDomains: ReadonlyArray<DisplayDomain> = isCurrentDeployment
    ? [...ctx.customDomains]
        .filter(
          (cd) => cd.environmentId === ctx.environmentId && cd.verificationStatus === "verified",
        )
        .sort((a, b) => a.domain.localeCompare(b.domain))
        .map((cd) => ({
          source: "custom" as const,
          id: cd.id,
          hostname: cd.domain,
          url: `https://${cd.domain}`,
          customDomain: cd,
        }))
    : [];

  // A verified custom domain also has a live frontline route. Exclude that route
  // even when the custom domain belongs to another environment; otherwise the
  // route would be misclassified as a generated platform alias.
  const verifiedCustomHostnames = new Set(
    ctx.customDomains
      .filter((domain) => domain.verificationStatus === "verified")
      .map((d) => d.domain),
  );

  const platformDisplayDomains: ReadonlyArray<DisplayDomain> = [...ctx.domains]
    .filter((d) => !verifiedCustomHostnames.has(d.fullyQualifiedDomainName))
    .sort((a, b) => a.fullyQualifiedDomainName.localeCompare(b.fullyQualifiedDomainName))
    .map((d) => ({
      source: "platform" as const,
      id: d.id,
      hostname: d.fullyQualifiedDomainName,
      url: `https://${d.fullyQualifiedDomainName}`,
      domain: d,
    }));

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
