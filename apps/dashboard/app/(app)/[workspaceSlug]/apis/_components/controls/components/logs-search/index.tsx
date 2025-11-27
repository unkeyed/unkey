import { useTRPC } from "@/lib/trpc/client";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { useMutation } from "@tanstack/react-query";
import { LLMSearch, toast } from "@unkey/ui";
import { useRef } from "react";
type Props = {
  apiList: ApiOverview[];
  onApiListChange: (apiList: ApiOverview[]) => void;
  onSearch: (value: boolean) => void;
};

export const LogsSearch = ({ onSearch, onApiListChange, apiList }: Props) => {
  const trpc = useTRPC();
  const originalApiList = useRef<ApiOverview[]>([]);
  const isSearchingRef = useRef<boolean>(false);
  const searchApiOverview = useMutation(
    trpc.api.overview.search.mutationOptions({
      onSuccess(data) {
        // Store original list before first search
        if (!isSearchingRef.current) {
          originalApiList.current = [...apiList];
          isSearchingRef.current = true;
        }
        onSearch(true);
        onApiListChange(data);
      },
      onError(error) {
        toast.error(error.message, {
          duration: 8000,
          position: "top-right",
          style: {
            whiteSpace: "pre-line",
          },
          className: "font-medium",
        });
      },
    }),
  );

  const handleClear = () => {
    // Reset to original state when search is cleared
    if (isSearchingRef.current && originalApiList.current.length > 0) {
      onApiListChange(originalApiList.current);
      isSearchingRef.current = false;
      onSearch(false);
    }
  };

  return (
    <LLMSearch
      exampleQueries={[
        "Show rate limited requests today",
        "Show requests that were not rate limited today",
        "Show requests in the last 5 minutes",
      ]}
      hideExplainer
      onClear={handleClear}
      placeholder="Search API using name or ID"
      isLoading={searchApiOverview.isPending}
      loadingText="Searching APIs..."
      searchMode="allowTypeDuringSearch"
      onSearch={(query) =>
        searchApiOverview.mutateAsync({
          query,
        })
      }
    />
  );
};
