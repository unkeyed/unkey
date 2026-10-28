"use client";

import { useFilters } from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/_overview/hooks/use-filters";
import { ListViews } from "@/components/list-views";
import { TimeRangeFilter } from "@/components/logs/datetime/time-range-filter";
import { ResourceSearchInput, useListSearch } from "@/components/resource-search-input";
import { StatsListCardSkeleton } from "@/components/stats-list-card/skeleton";
import { GRID_PAGE_SIZE } from "@/lib/list-limits";
import { searchedListOf, useInfiniteList } from "@/lib/loadable/list";
import { useNamespaces } from "@/lib/queries/ratelimit-namespaces";
import { useBatchRatelimitTimeseries } from "./hooks/use-batch-timeseries";
import { NAMESPACE_TABLE, NamespaceCard, NamespaceRow } from "./list/namespace-list-item";
import { NamespacesEmpty } from "./list/namespaces-empty";

export function NamespaceListClient() {
  const search = useListSearch();
  const { filters, updateFilters } = useFilters();
  const list = searchedListOf(
    useInfiniteList(useNamespaces({ search, pageSize: GRID_PAGE_SIZE }), (page) => page.namespaces),
    search,
  );

  const namespaceIds = list.status === "ready" ? list.rows.map((ns) => ns.id) : [];
  const { timeseriesByNamespace, isLoading, isError } = useBatchRatelimitTimeseries(namespaceIds);
  const statsOf = (namespace: { id: string; name: string }) => ({
    namespace,
    timeseries: timeseriesByNamespace[namespace.id],
    isLoading,
    isError,
  });

  return (
    <ListViews
      list={list}
      viewStorageKey="namespaces-view"
      noun="namespaces"
      failedTitle="Could not load namespaces"
      empty={<NamespacesEmpty />}
      controls={
        <>
          <ResourceSearchInput label="Search namespaces" placeholder="Search namespaces" />
          <TimeRangeFilter filters={filters} updateFilters={updateFilters} look="list" />
        </>
      }
      table={NAMESPACE_TABLE}
      rowKey={(namespace) => namespace.id}
      renderCard={(namespace) => <NamespaceCard {...statsOf(namespace)} />}
      renderRow={(namespace) => <NamespaceRow {...statsOf(namespace)} />}
      cardSkeleton={<StatsListCardSkeleton />}
    />
  );
}
