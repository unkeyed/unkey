"use client";

import { ListViews } from "@/components/list-views";
import { TimeRangeFilter } from "@/components/logs/datetime/time-range-filter";
import { ResourceSearchInput, useListSearch } from "@/components/resource-search-input";
import { StatsListCardSkeleton } from "@/components/stats-list-card/skeleton";
import { GRID_PAGE_SIZE } from "@/lib/list-limits";
import { searchedListOf, useInfiniteList } from "@/lib/loadable/list";
import { trpc } from "@/lib/trpc/client";
import { useSearchParams } from "next/navigation";
import { ApiCard, ApiRow, KEYSPACE_TABLE } from "./api-list-item";
import { EmptyKeyspaces } from "./empty-keyspaces";
import { useFilters } from "./hooks/use-filters";

export function ApiListClient({ workspaceSlug }: { workspaceSlug: string }) {
  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";
  const search = useListSearch();
  const { filters, updateFilters } = useFilters();

  const query = trpc.api.overview.query.useInfiniteQuery(
    { limit: GRID_PAGE_SIZE, search: search || undefined },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
      keepPreviousData: true,
    },
  );
  const list = searchedListOf(
    useInfiniteList(query, (page) => page.apiList),
    search,
  );

  return (
    <ListViews
      list={list}
      viewStorageKey="keyspaces-view"
      noun="keyspaces"
      failedTitle="Could not load keyspaces"
      empty={<EmptyKeyspaces workspaceSlug={workspaceSlug} isNewApi={isNewApi} />}
      controls={
        <>
          <ResourceSearchInput
            label="Search keyspaces"
            placeholder="Search keyspaces by name or ID"
          />
          <TimeRangeFilter filters={filters} updateFilters={updateFilters} look="list" />
        </>
      }
      table={KEYSPACE_TABLE}
      rowKey={(api) => api.id}
      renderCard={(api) => <ApiCard api={api} />}
      renderRow={(api) => <ApiRow api={api} />}
      cardSkeleton={<StatsListCardSkeleton subtitle />}
    />
  );
}
