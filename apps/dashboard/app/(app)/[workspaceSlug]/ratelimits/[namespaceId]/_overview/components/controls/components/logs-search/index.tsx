import { useTRPC } from "@/lib/trpc/client";
import { LLMSearch, toast, transformStructuredOutputToFilters } from "@unkey/ui";
import { useFilters } from "../../../../hooks/use-filters";

import { useMutation } from "@tanstack/react-query";

export const LogsSearch = () => {
  const trpc = useTRPC();
  const { filters, updateFilters } = useFilters();
  const queryLLMForStructuredOutput = useMutation(trpc.ratelimit.logs.ratelimitLlmSearch.mutationOptions({
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
      const errorMessage = `Unable to process your search request${error.message ? `' ${error.message} '` : "."
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
  }));

  return (
    <LLMSearch
      exampleQueries={[
        "Show all identifiers that is test",
        "Show all identifiers that include cust_",
        "Show all identifiers for customer in the last 1h",
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
