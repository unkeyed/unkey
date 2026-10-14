import type { CustomDomain } from "@/lib/collections/deploy/custom-domains";
import type { Domain } from "@/lib/collections/deploy/domains";
import type { Environment } from "@/lib/collections/deploy/environments";
import {
  type DisplayDomain,
  customDisplayDomain,
  listedPlatformDomains,
  platformDisplayDomain,
} from "../../components/display-domain";

export function domainsPageList(
  domains: ReadonlyArray<Domain>,
  customDomains: ReadonlyArray<CustomDomain>,
): DisplayDomain[] {
  return [
    ...customDomains.map(customDisplayDomain),
    ...listedPlatformDomains(domains, customDomains).map(platformDisplayDomain),
  ];
}

type DomainEnvironment = Pick<Environment, "id" | "slug" | "kind">;

export type DomainListItem = {
  domain: DisplayDomain;
  environment: DomainEnvironment | undefined;
};

function withEnvironments(
  list: ReadonlyArray<DisplayDomain>,
  environments: ReadonlyArray<DomainEnvironment>,
): DomainListItem[] {
  const byId = new Map(environments.map((e) => [e.id, e]));
  return list.map((domain) => ({ domain, environment: byId.get(domain.environmentId) }));
}

function filterDomains(
  items: ReadonlyArray<DomainListItem>,
  search: string,
): ReadonlyArray<DomainListItem> {
  const query = search.trim().toLowerCase();
  if (!query) {
    return items;
  }
  return items.filter(
    ({ domain, environment }) =>
      domain.hostname.toLowerCase().includes(query) ||
      (environment?.slug.toLowerCase().includes(query) ?? false),
  );
}

export type DomainsView =
  | { state: "loading" }
  | { state: "failed" }
  | { state: "empty" }
  | { state: "no-match"; search: string }
  | { state: "list"; domains: ReadonlyArray<DomainListItem>; hasMore: boolean };

export function domainsView({
  isLoading,
  failed,
  domains,
  customDomains,
  environments,
  search,
  limit,
}: {
  isLoading: boolean;
  failed: boolean;
  domains: ReadonlyArray<Domain>;
  customDomains: ReadonlyArray<CustomDomain>;
  environments: ReadonlyArray<DomainEnvironment>;
  search: string;
  limit: number;
}): DomainsView {
  if (isLoading) {
    return { state: "loading" };
  }
  if (failed) {
    return { state: "failed" };
  }
  const list = domainsPageList(domains, customDomains);
  if (list.length === 0) {
    return { state: "empty" };
  }
  const matches = filterDomains(withEnvironments(list, environments), search);
  if (matches.length === 0) {
    return { state: "no-match", search: search.trim() };
  }
  return {
    state: "list",
    domains: matches.slice(0, limit),
    hasMore: matches.length > limit,
  };
}
