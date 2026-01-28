import { trpc } from "@/lib/trpc/client";
import { LLMSearch, toast, transformStructuredOutputToFilters } from "@unkey/ui";
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
            position: "top-right",
            style: {
              whiteSpace: "pre-line",
            },
          },
        );
        return;
      }
      type ValidField = (typeof VALID_FIELDS)[number];

      const transformedFilters = transformStructuredOutputToFilters(
        data as {
          filters: Array<{
            field: string;
            filters: Array<{ operator: string; value: string | number }>;
          }>;
        },
        filters,
      ) as typeof filters;

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
