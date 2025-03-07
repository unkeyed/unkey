import { LogsLLMSearch } from "@/components/logs/llm-search";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import type { ApiOverview } from "@/lib/trpc/routers/api/overview/query-overview/schemas";
import { useRef } from "react";
type Props = {
  apiList: ApiOverview[];
  onApiListChange: (apiList: ApiOverview[]) => void;
  onSearch: (value: boolean) => void;
};

export const LogsSearch = ({ onSearch, onApiListChange, apiList }: Props) => {
  const originalApiList = useRef<ApiOverview[]>([]);
  const isSearchingRef = useRef<boolean>(false);
  const searchApiOverview = trpc.api.overview.search.useMutation({
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
        important: true,
        position: "top-right",
        style: {
          whiteSpace: "pre-line",
        },
        className: "font-medium",
      });
    },
  });

  const handleClear = () => {
    // Reset to original state when search is cleared
    if (isSearchingRef.current && originalApiList.current.length > 0) {
      onApiListChange(originalApiList.current);
      isSearchingRef.current = false;
      onSearch(false);
    }
  };

  return (
    <LogsLLMSearch
      hideExplainer
      onClear={handleClear}
      placeholder="Search API using name or ID"
      isLoading={searchApiOverview.isLoading}
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
