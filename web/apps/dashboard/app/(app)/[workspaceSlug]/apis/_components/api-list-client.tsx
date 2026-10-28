"use client";

import { ListPager } from "@/components/list-pager";
import { LoadError } from "@/components/load-error";
import { StatsListCardSkeleton } from "@/components/stats-list-card/skeleton";
import { useInfiniteList } from "@/lib/loadable/list";
import { trpc } from "@/lib/trpc/client";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { IconNodesOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";
import {
  EmptyState,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
  ResourceListContent,
} from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import { useState } from "react";
import { ApiListCard } from "./api-list-card";
import { ApiListControls } from "./controls";
import { EmptyKeyspaces } from "./empty-keyspaces";

const DEFAULT_LIMIT = 10;
const SKELETON_COUNT = 3;

export const ApiListClient = ({ workspaceSlug }: { workspaceSlug: string }) => {
  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";

  const query = trpc.api.overview.query.useInfiniteQuery(
    { limit: DEFAULT_LIMIT },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
    },
  );
  const view = useInfiniteList(query, (page) => page.apiList);
  const total = query.data?.pages[0]?.total ?? 0;

  const [searchResults, setSearchResults] = useState<ApiOverview[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const rows = view.status === "ready" ? view.rows : [];

  return (
    <div className="flex flex-col gap-4">
      {view.status !== "empty" && (
        <ApiListControls
          apiList={rows}
          onApiListChange={setSearchResults}
          onSearch={setIsSearching}
        />
      )}

      {match(view)
        .with({ status: "loading" }, () => (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5 w-full">
            {Array.from({ length: SKELETON_COUNT }).map((_, i) => (
              // biome-ignore lint/suspicious/noArrayIndexKey: It's okay to use index
              <StatsListCardSkeleton key={i} subtitle />
            ))}
          </div>
        ))
        .with({ status: "failed" }, ({ error, retry }) => (
          <LoadError title="Could not load keyspaces" error={error} onRetry={retry} />
        ))
        .with({ status: "empty" }, () => (
          <EmptyKeyspaces workspaceSlug={workspaceSlug} isNewApi={isNewApi} />
        ))
        .with({ status: "ready" }, ({ footer }) => {
          const shown = isSearching ? searchResults : rows;
          if (shown.length === 0) {
            return (
              <ResourceListContent>
                <EmptyState frame="none">
                  <EmptyStateIcon>
                    <IconNodesOutline18 />
                  </EmptyStateIcon>
                  <EmptyStateHeader>
                    <EmptyStateTitle>No keyspaces found</EmptyStateTitle>
                    <EmptyStateDescription>
                      No keyspaces match your search criteria. Try a different search term.
                    </EmptyStateDescription>
                  </EmptyStateHeader>
                </EmptyState>
              </ResourceListContent>
            );
          }
          return (
            <>
              <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5 w-full">
                {shown.map((api) => (
                  <ApiListCard api={api} key={api.id} />
                ))}
              </div>

              {!isSearching && (
                <div className="flex flex-col items-center justify-center mt-8 pb-8 gap-4">
                  {footer.kind === "more" || footer.kind === "loadingMore" ? (
                    <div className="text-center text-sm text-gray-11">
                      Showing {rows.length} of {total} keyspaces
                    </div>
                  ) : null}
                  <ListPager
                    state={footer}
                    noun="keyspaces"
                    className="justify-center border-t-0 p-0"
                  />
                </div>
              )}
            </>
          );
        })
        .exhaustive()}
    </div>
  );
};
