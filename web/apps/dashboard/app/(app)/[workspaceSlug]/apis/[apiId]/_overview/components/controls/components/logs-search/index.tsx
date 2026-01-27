import { trpc } from "@/lib/trpc/client";
import { LLMSearch, toast, transformStructuredOutputToFilters } from "@unkey/ui";
import { useFilters } from "../../../../hooks/use-filters";

export const LogsSearch = ({ apiId }: { apiId: string }) => {
  const { filters, updateFilters } = useFilters();
  const queryLLMForStructuredOutput = trpc.api.keys.llmSearch.useMutation({
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
      const transformedFilters = transformStructuredOutputToFilters(
        data as {
          filters: Array<{
            field: string;
            filters: Array<{ operator: string; value: string | number }>;
          }>;
        },
        filters,
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
        "Show rate limited outcomes",
        "Show identity that starts with test_",
        "Show name that starts with test_ in the last hour",
      ]}
      isLoading={queryLLMForStructuredOutput.isLoading}
      searchMode="manual"
      onSearch={(query) =>
        queryLLMForStructuredOutput.mutateAsync({
          apiId,
          query,
          timestamp: Date.now(),
        })
      }
    />
  );
};
