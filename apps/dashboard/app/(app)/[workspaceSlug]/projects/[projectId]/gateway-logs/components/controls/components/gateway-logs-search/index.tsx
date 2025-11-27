import { useTRPC } from "@/lib/trpc/client";
import { LLMSearch, toast, transformStructuredOutputToFilters } from "@unkey/ui";
import { useGatewayLogsFilters } from "../../../../hooks/use-gateway-logs-filters";

import { useMutation } from "@tanstack/react-query";

export const GatewayLogsSearch = () => {
  const trpc = useTRPC();
  const { filters, updateFilters } = useGatewayLogsFilters();
  const queryLLMForStructuredOutput = useMutation(
    trpc.logs.llmSearch.mutationOptions({
      onSuccess(data) {
        if (data?.filters.length === 0 || !data) {
          toast.error(
            "Please provide more specific search criteria. Your query requires additional details for accurate results.",
            {
              duration: 8000,
              position: "top-right",
              style: {
                whiteSpace: "pre-line",
              },
            },
          );
          return;
        }
        const transformedFilters = transformStructuredOutputToFilters(data, filters);
        updateFilters(transformedFilters);
      },
      onError(error) {
        const errorMessage = `Unable to process your search request${
          error.message ? `' ${error.message} '` : "."
        } Please try again or refine your search criteria.`;

        toast.error(errorMessage, {
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

  return (
    <LLMSearch
      exampleQueries={[
        "Show failed requests today",
        "Show auth errors in the last 3h",
        "Show API calls from a path that includes api/v1/",
      ]}
      isLoading={queryLLMForStructuredOutput.isPending}
      searchMode="manual"
      onSearch={(query) =>
        queryLLMForStructuredOutput.mutateAsync({
          query,
          timestamp: Date.now(),
        })
      }
    />
  );
};
