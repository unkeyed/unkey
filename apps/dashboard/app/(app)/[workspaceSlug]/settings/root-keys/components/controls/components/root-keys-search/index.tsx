import { transformStructuredOutputToFilters } from "@/components/logs/validation/utils/transform-structured-output-filter-format";
import { trpc } from "@/lib/trpc/client";
import { LLMSearch, toast } from "@unkey/ui";
import { useFilters } from "../../../../hooks/use-filters";

export const RootKeysSearch = () => {
  const { filters, updateFilters } = useFilters();

  const queryLLMForStructuredOutput = trpc.settings.rootKeys.llmSearch.useMutation({
    onSuccess(data) {
      if (!data?.filters?.length) {
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
        error.message ? `: ${error.message}` : "."
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
        "Show keys with api.read permissions",
        "Show keys containing database permissions",
        "Find keys named exactly 'super_admin'",
        "Show keys with write permissions and user keys",
      ]}
      isLoading={queryLLMForStructuredOutput.isLoading}
      searchMode="manual"
      onSearch={(query) =>
        queryLLMForStructuredOutput.mutateAsync({
          query,
        })
      }
    />
  );
};
