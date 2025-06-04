import { LogsLLMSearch } from "@/components/logs/llm-search";
import { transformStructuredOutputToFilters } from "@/components/logs/validation/utils/transform-structured-output-filter-format";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import type { KeyDetailsFilterValue } from "../../../../filters.schema";
import { useFilters } from "../../../../hooks/use-filters";

const VALID_FIELDS = ["startTime", "endTime", "since", "outcomes"] as const;
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
      type ValidField = (typeof VALID_FIELDS)[number];

      const transformedFilters = transformStructuredOutputToFilters(data, filters);

      const validFilters = transformedFilters.filter((filter): filter is KeyDetailsFilterValue =>
        VALID_FIELDS.includes(filter.field as ValidField),
      );

      updateFilters(validFilters);
    },
    onError(error) {
      const errorMessage = `Unable to process your search request${
        error.message ? `' ${error.message} '` : "."
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
      exampleQueries={["Show rate limited outcomes"]}
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
