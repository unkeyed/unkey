import { useFilters } from "@/app/(app)/logs/hooks/use-filters";
import { LogsLLMSearch } from "@/components/logs/llm-search";
import { transformStructuredOutputToFilters } from "@/components/logs/validation/utils/transform-structured-output-filter-format";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";

export const LogsSearch = () => {
  const { filters, updateFilters } = useFilters();
  const queryLLMForStructuredOutput = trpc.logs.llmSearch.useMutation({
    onSuccess(data) {
      if (data?.filters.length === 0 || !data) {
        toast.error(
          "Please provide more specific search criteria. Your query requires additional details for accurate results.",
          {
            duration: 8000,
            important: true,
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
        error.message ? "' ${error.message} '" : "."
      } Please try again or refine your search criteria.`;

      toast.error(errorMessage, {
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

  return (
    <LogsLLMSearch
      exampleQueries={[
        "Show failed requests today",
        "Show auth errors in the last 3h",
        "Show API calls from a path that includes api/v1/",
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
