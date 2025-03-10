import { LogsLLMSearch } from "@/components/logs/llm-search";
import { transformStructuredOutputToFilters } from "@/components/logs/validation/utils/transform-structured-output-filter-format";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
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
      updateFilters(transformedFilters as any);
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
