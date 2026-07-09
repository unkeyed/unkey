"use client";

import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Button, Empty } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { type PropsWithChildren, useEffect, useMemo, useState } from "react";
import { ApiListCard } from "./api-list-card";
import { ApiListControls } from "./controls";
import { EmptyKeyspaces } from "./empty-keyspaces";
import { ApiCardSkeleton } from "./skeleton";

const DEFAULT_LIMIT = 10;
const SKELETON_COUNT = 3;

export const ApiListClient = ({ workspaceSlug }: { workspaceSlug: string }) => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";

  const {
    data: apisData,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = trpc.api.overview.query.useInfiniteQuery(
    { limit: DEFAULT_LIMIT },
    {
      getNextPageParam: (lastPage) => lastPage.nextCursor,
    },
  );

  const allApis = useMemo(() => {
    if (!apisData?.pages) {
      return [];
    }
    return apisData.pages.flatMap((page) => page.apiList);
  }, [apisData]);

  const [apiList, setApiList] = useState(allApis);
  const [isSearching, setIsSearching] = useState(false);

  useEffect(() => {
    setApiList(allApis);
  }, [allApis]);

  useEffect(() => {
    if (error) {
      router.push(routes.workspaces.create());
    }
  }, [error, router]);

  const loadMore = () => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  };

  const hasNoKeyspaces = !isLoading && allApis.length === 0;

  return (
    <div className="flex flex-col gap-4">
      {!hasNoKeyspaces && (
        <ApiListControls apiList={allApis} onApiListChange={setApiList} onSearch={setIsSearching} />
      )}

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5 w-full">
          {Array.from({ length: SKELETON_COUNT }).map((_, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: It's okay to use index
            <ApiCardSkeleton key={i} />
          ))}
        </div>
      ) : hasNoKeyspaces ? (
        <EmptyKeyspaces workspaceSlug={workspaceSlug} isNewApi={isNewApi} />
      ) : apiList.length > 0 ? (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5 w-full">
            {apiList.map((api) => (
              <ApiListCard api={api} key={api.id} />
            ))}
          </div>

          {!isSearching && hasNextPage && (
            <div className="flex flex-col items-center justify-center mt-8 pb-8 gap-4">
              <div className="text-center text-sm text-accent-11">
                Showing {apiList.length} of {apisData?.pages[0]?.total || 0} keyspaces
              </div>

              <Button onClick={loadMore} disabled={isFetchingNextPage} size="md">
                {isFetchingNextPage ? (
                  <div className="flex flex-row items-center gap-2">
                    <div className="animate-spin h-4 w-4 border-2 border-gray-7 border-t-transparent rounded-full" />
                    <span>Loading...</span>
                  </div>
                ) : (
                  <div className="flex flex-row items-center gap-2">
                    <span>Load more</span>
                  </div>
                )}
              </Button>
            </div>
          )}
        </>
      ) : (
        <EmptyComponentSpacer>
          <Empty className="m-0 p-0">
            <Empty.Icon />
            <Empty.Title>No keyspaces found</Empty.Title>
            <Empty.Description>
              No keyspaces match your search criteria. Try a different search term.
            </Empty.Description>
          </Empty>
        </EmptyComponentSpacer>
      )}
    </div>
  );
};

const EmptyComponentSpacer = ({ children }: PropsWithChildren) => {
  return (
    <div className="h-full min-h-[300px] flex items-center justify-center">
      <div className="flex justify-center items-center">{children}</div>
    </div>
  );
};
