import { LogsLLMSearch } from "@/components/logs/llm-search";
import { transformStructuredOutputToFilters } from "@/components/logs/validation/utils/transform-structured-output-filter-format";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { useFilters } from "../../../../hooks/use-filters";

export const LogsSearch = ({ keyspaceId }: { keyspaceId: string }) => {
  const { filters, updateFilters } = useFilters();
  const queryLLMForStructuredOutput = trpc.api.keys.listLlmSearch.useMutation({
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
        error.message ? `: ${error.message}` : "."
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
        "Find key exactly key_abc123xyz",
        "Show keys with ID containing 'test'",
        "Show keys with name 'Temp Key' and ID starting with 'temp_'",
        "Find keys where identity is 'dev_user' or name contains 'debug'",
      ]}
      isLoading={queryLLMForStructuredOutput.isLoading}
      searchMode="manual"
      onSearch={(query) =>
        queryLLMForStructuredOutput.mutateAsync({
          keyspaceId,
          query,
        })
      }
    />
  );
};
