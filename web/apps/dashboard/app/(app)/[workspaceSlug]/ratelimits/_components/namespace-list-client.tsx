"use client";

import { ListViews } from "@/components/list-views";
import { LoadError } from "@/components/load-error";
import { useListSearch } from "@/components/resource-search-input";
import { StatsListCardSkeleton } from "@/components/stats-list-card/skeleton";
import { useListView } from "@/hooks/use-list-view";
import { GRID_PAGE_SIZE } from "@/lib/list-limits";
import { searchedListOf, useInfiniteList } from "@/lib/loadable/list";
import { useNamespaces } from "@/lib/queries/ratelimit-namespaces";
import { match } from "@unkey/match";
import { ResourceList } from "@unkey/ui";
import { NamespaceListControls } from "./controls";
import { useBatchRatelimitTimeseries } from "./hooks/use-batch-timeseries";
import { NAMESPACE_TABLE, NamespaceCard, NamespaceRow } from "./list/namespace-list-item";
import { NamespacesEmpty } from "./list/namespaces-empty";

export function NamespaceListClient() {
  const [view, setView] = useListView("namespaces-view");
  const search = useListSearch();
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

  return match(list)
    .with({ status: "empty" }, () => <NamespacesEmpty />)
    .otherwise((content) => (
      <ResourceList>
        <NamespaceListControls view={view} onViewChange={setView} />
        {match(content)
          .with({ status: "failed" }, ({ error, retry }) => (
            <LoadError title="Could not load namespaces" error={error} onRetry={retry} />
          ))
          .otherwise((shown) => (
            <ListViews
              view={view}
              content={shown}
              noun="namespaces"
              table={NAMESPACE_TABLE}
              rowKey={(namespace) => namespace.id}
              renderCard={(namespace) => <NamespaceCard {...statsOf(namespace)} />}
              renderRow={(namespace, table) => (
                <NamespaceRow table={table} {...statsOf(namespace)} />
              )}
              cardSkeleton={<StatsListCardSkeleton />}
            />
          ))}
      </ResourceList>
    ));
}
