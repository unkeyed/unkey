import { LogsLLMSearch } from "@/components/logs/llm-search";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";

export const LogsSearch = () => {
  const searchNamespace = trpc.ratelimit.namespace.search.useMutation({
    onSuccess() {
      return {};
    },
    onError(error) {
      toast.error(error.message, {
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

  // const handleClear = () => {
  //   setNamespaces(initialNamespaces);
  // };

  return (
    <LogsLLMSearch
      hideExplainer
      // onClear={handleClear}
      placeholder="Search API"
      isLoading={searchNamespace.isLoading}
      onSearch={(query) =>
        searchNamespace.mutateAsync({
          query,
        })
      }
    />
  );
};
