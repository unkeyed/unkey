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

export function filterDomains(
  list: ReadonlyArray<DisplayDomain>,
  search: string,
  environments: ReadonlyArray<Pick<Environment, "id" | "slug">>,
): ReadonlyArray<DisplayDomain> {
  const query = search.trim().toLowerCase();
  if (!query) {
    return list;
  }
  const slugs = new Map(environments.map((e) => [e.id, e.slug.toLowerCase()]));
  return list.filter(
    (d) =>
      d.hostname.toLowerCase().includes(query) ||
      (slugs.get(d.environmentId)?.includes(query) ?? false),
  );
}

export type DomainsView =
  | { state: "loading" }
  | { state: "failed" }
  | { state: "empty" }
  | { state: "no-match"; search: string }
  | { state: "list"; domains: ReadonlyArray<DisplayDomain>; hasMore: boolean };

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
  environments: ReadonlyArray<Pick<Environment, "id" | "slug">>;
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
  const matches = filterDomains(list, search, environments);
  if (matches.length === 0) {
    return { state: "no-match", search: search.trim() };
  }
  return {
    state: "list",
    domains: matches.slice(0, limit),
    hasMore: matches.length > limit,
  };
}
