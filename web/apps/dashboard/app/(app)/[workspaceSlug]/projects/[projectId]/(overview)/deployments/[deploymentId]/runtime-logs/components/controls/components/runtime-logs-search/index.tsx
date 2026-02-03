"use client";

import { trpc } from "@/lib/trpc/client";
import { LLMSearch, toast, transformStructuredOutputToFilters } from "@unkey/ui";
import { useRuntimeLogsFilters } from "../../../../hooks/use-runtime-logs-filters";

export const RuntimeLogsSearch = () => {
  const { filters, updateFilters } = useRuntimeLogsFilters();
  const queryLLMForStructuredOutput = trpc.deploy.runtimeLogs.llmSearch.useMutation({
    onSuccess(data) {
      const typedData = data as
        | {
            filters: Array<{
              field: string;
              filters: Array<{ operator: string; value: string | number }>;
            }>;
          }
        | undefined;

      if (!typedData || typedData.filters.length === 0) {
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
      const transformedFilters = transformStructuredOutputToFilters(
        typedData,
        filters.filter((f) => f.field !== "message"),
      ) as typeof filters;
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
  });

  return (
    <LLMSearch
      exampleQueries={[
        "Show errors in the last hour",
        "Show warnings containing 'timeout'",
        "Show all debug logs from yesterday",
      ]}
      isLoading={queryLLMForStructuredOutput.isLoading}
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
