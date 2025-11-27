import { useTRPC } from "@/lib/trpc/client";
import { LLMSearch, toast, transformStructuredOutputToFilters } from "@unkey/ui";
import { useFilters } from "../../../../hooks/use-filters";

import { useMutation } from "@tanstack/react-query";

export const PermissionSearch = () => {
  const trpc = useTRPC();
  const { filters, updateFilters } = useFilters();

  const queryLLMForStructuredOutput = useMutation(trpc.authorization.permissions.llmSearch.mutationOptions({
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
  }));

  return (
    <LLMSearch
      exampleQueries={[
        "find admin permissions",
        "show api.read and api.write permissions",
        "permissions for database access",
        "show permissions ending with .create",
        "find permissions starting with api_",
        "permissions for role_123",
      ]}
      isLoading={queryLLMForStructuredOutput.isPending}
      searchMode="manual"
      onSearch={(query) =>
        queryLLMForStructuredOutput.mutateAsync({
          query,
        })
      }
    />
  );
};
