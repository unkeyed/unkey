import { transformStructuredOutputToFilters } from "@/components/logs/validation/utils/transform-structured-output-filter-format";
import { trpc } from "@/lib/trpc/client";
import { LLMSearch, toast } from "@unkey/ui";
import { useFilters } from "../../../../hooks/use-filters";

export const DeploymentListSearch = () => {
  const { filters, updateFilters } = useFilters();

  const queryLLMForStructuredOutput = trpc.deployment.search.useMutation({
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
      const errorMessage = `Unable to process your search request${error.message ? `: ${error.message}` : "."
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
        "show failed deployments",
        "production deployments",
        "deployments from main branch",
        "recent deployments",
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
