import type {
  ApiOverview,
  ApisOverviewResponse,
} from "@/lib/trpc/routers/api/query-overview/schemas";
import { ChevronDown } from "@unkey/icons";
import { Button } from "@unkey/ui";
import type { Dispatch, SetStateAction } from "react";
import { ApiListCard } from "./api-list-card";
import { useFetchApiOverview } from "./hooks/use-fetch-api-overview";

export const ApiListGrid = ({
  initialData,
  setApiList,
  apiList,
  isSearching,
}: {
  initialData: ApisOverviewResponse;
  apiList: ApiOverview[];
  setApiList: Dispatch<SetStateAction<ApiOverview[]>>;
  isSearching?: boolean;
}) => {
  const { total, loadMore, isLoading, hasMore } = useFetchApiOverview(initialData, setApiList);

  return (
    <>
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5 gap-3 md:gap-5 w-full">
        {apiList.map((api) => (
          <ApiListCard api={api} key={api.id} />
        ))}
      </div>

      <div className="flex flex-col items-center justify-center mt-8 space-y-4">
        <div className="text-center text-sm text-accent-11">
          Showing {apiList.length} of {total} APIs
        </div>

        {!isSearching && hasMore && (
          <Button onClick={loadMore} disabled={isLoading}>
            {isLoading ? (
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
