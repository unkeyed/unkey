"use client";

import { ListViews } from "@/components/list-views";
import { LoadError } from "@/components/load-error";
import { useListSearch } from "@/components/resource-search-input";
import { StatsListCardSkeleton } from "@/components/stats-list-card/skeleton";
import { useListView } from "@/hooks/use-list-view";
import { GRID_PAGE_SIZE } from "@/lib/list-limits";
import { searchedListOf, useInfiniteList } from "@/lib/loadable/list";
import { trpc } from "@/lib/trpc/client";
import { match } from "@unkey/match";
import { ResourceList } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { ApiCard, ApiRow, KEYSPACE_TABLE } from "./api-list-item";
import { ApiListControls } from "./controls";
import { EmptyKeyspaces } from "./empty-keyspaces";

export function ApiListClient({ workspaceSlug }: { workspaceSlug: string }) {
  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";
  const [view, setView] = useListView("keyspaces-view");
  const search = useListSearch();

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

  return match(list)
    .with({ status: "empty" }, () => (
      <EmptyKeyspaces workspaceSlug={workspaceSlug} isNewApi={isNewApi} />
    ))
    .otherwise((content) => (
      <ResourceList>
        <ApiListControls view={view} onViewChange={setView} />
        {match(content)
          .with({ status: "failed" }, ({ error, retry }) => (
            <LoadError title="Could not load keyspaces" error={error} onRetry={retry} />
          ))
          .otherwise((shown) => (
            <ListViews
              view={view}
              content={shown}
              noun="keyspaces"
              table={KEYSPACE_TABLE}
              rowKey={(api) => api.id}
              renderCard={(api) => <ApiCard api={api} />}
              renderRow={(api, table) => <ApiRow api={api} table={table} />}
              cardSkeleton={<StatsListCardSkeleton subtitle />}
            />
          ))}
      </ResourceList>
    ));
}
