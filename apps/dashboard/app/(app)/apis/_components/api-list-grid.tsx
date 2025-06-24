import { EmptyComponentSpacer } from "@/components/empty-component-spacer";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { ChevronDown } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import { ApiListCard } from "./api-list-card";

export const ApiListGrid = ({
  apiList,
  isSearching,
  onLoadMore,
  hasMore,
  isLoadingMore,
  total,
}: {
  apiList: ApiOverview[];
  isSearching?: boolean;
  onLoadMore: () => void;
  hasMore: boolean;
  isLoadingMore: boolean;
  total: number;
}) => {
  if (apiList.length === 0) {
    return (
      <EmptyComponentSpacer>
        <Empty className="m-0 p-0">
          <Empty.Icon />
          <Empty.Title>No APIs found</Empty.Title>
          <Empty.Description>
            No APIs match your search criteria. Try a different search term.
          </Empty.Description>
        </Empty>
      </EmptyComponentSpacer>
    );
  }

  return (
    <>
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-5 w-full p-5">
        {apiList.map((api) => (
          <ApiListCard api={api} key={api.id} />
        ))}
      </div>

      <div className="flex flex-col items-center justify-center mt-8 space-y-4 pb-8">
        <div className="text-center text-sm text-accent-11">
          Showing {apiList.length} of {total} APIs
        </div>

        {!isSearching && hasMore && (
          <Button onClick={onLoadMore} disabled={isLoadingMore} size="md">
            {isLoadingMore ? (
              <div className="flex items-center space-x-2">
                <div className="animate-spin h-4 w-4 border-2 border-gray-7 border-t-transparent rounded-full" />
                <span>Loading...</span>
              </div>
            ) : (
              <div className="flex items-center space-x-2">
                <ChevronDown />
                <span>Load more</span>
              </div>
            )}
          </Button>
        )}
      </div>
    </>
  );
};
