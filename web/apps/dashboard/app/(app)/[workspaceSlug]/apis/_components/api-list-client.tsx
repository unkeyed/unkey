"use client";

import { EmptyComponentSpacer } from "@/components/empty-component-spacer";
import { trpc } from "@/lib/trpc/client";
import { BookBookmark } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useMemo, useState } from "react";
import { ApiListCard } from "./api-list-card";
import { ApiListControlCloud } from "./control-cloud";
import { ApiListControls } from "./controls";
import { CreateApiButton } from "./create-api-button";
import { ApiCardSkeleton } from "./skeleton";

const DEFAULT_LIMIT = 10;

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
      router.push("/new");
    }
  }, [error, router]);

  const loadMore = () => {
    if (hasNextPage && !isFetchingNextPage) {
      fetchNextPage();
    }
  };

  return (
    <div className="flex flex-col">
      <ApiListControls apiList={allApis} onApiListChange={setApiList} onSearch={setIsSearching} />
      <ApiListControlCloud />

      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-5 w-full p-5">
          {Array.from({ length: DEFAULT_LIMIT }).map((_, i) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: It's okay to use index
            <ApiCardSkeleton key={i} />
          ))}
        </div>
      ) : apiList.length > 0 ? (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-5 w-full p-5">
            {apiList.map((api) => (
              <ApiListCard api={api} key={api.id} />
            ))}
          </div>

          <div className="flex flex-col items-center justify-center mt-8 space-y-4 pb-8">
            <div className="text-center text-sm text-accent-11">
              Showing {apiList.length} of {apisData?.pages[0]?.total || 0} APIs
            </div>

            {!isSearching && hasNextPage && (
              <Button onClick={loadMore} disabled={isFetchingNextPage} size="md">
                {isFetchingNextPage ? (
                  <div className="flex items-center space-x-2">
                    <div className="animate-spin h-4 w-4 border-2 border-gray-7 border-t-transparent rounded-full" />
                    <span>Loading...</span>
                  </div>
                ) : (
                  <div className="flex items-center space-x-2">
                    <span>Load more</span>
                  </div>
                )}
              </Button>
            )}
          </div>
        </>
      ) : (
        <EmptyComponentSpacer>
          <Empty className="m-0 p-0">
            <Empty.Icon />
            <Empty.Title>No APIs found</Empty.Title>
            <Empty.Description>
              {isSearching
                ? "No APIs match your search criteria. Try a different search term."
                : "You haven't created any APIs yet. Create one to get started."}
            </Empty.Description>
            {!isSearching && (
              <Empty.Actions className="mt-4">
                <CreateApiButton defaultOpen={isNewApi} workspaceSlug={workspaceSlug} />
                <a
                  href="https://www.unkey.com/docs/introduction"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <Button size="md">
                    <BookBookmark />
                    Documentation
                  </Button>
                </a>
              </Empty.Actions>
            )}
          </Empty>
        </EmptyComponentSpacer>
      )}
    </div>
  );
};
