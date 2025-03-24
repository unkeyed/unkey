import { LogsLLMSearch } from "@/components/logs/llm-search";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { useRef } from "react";

type LogsSearchProps = {
  setNamespaces: (namespaces: { id: string; name: string }[]) => void;
  initialNamespaces: { id: string; name: string }[];
};

export const LogsSearch = ({ setNamespaces, initialNamespaces }: LogsSearchProps) => {
  const isSearchingRef = useRef<boolean>(false);

  const searchNamespace = trpc.ratelimit.namespace.search.useMutation({
    onSuccess(data) {
      if (!isSearchingRef.current) {
        isSearchingRef.current = true;
      }
      setNamespaces(data);
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

  const handleClear = () => {
    // Only reset if we have performed a search
    if (isSearchingRef.current) {
      setNamespaces(initialNamespaces);
      isSearchingRef.current = false;
    }
  };

  return (
    <LogsLLMSearch
      exampleQueries={[
        "Show failed requests today",
        "Show passed requests from the last 1 hour",
        "Show failed requests that includes cust_ in the identifier",
      ]}
      hideExplainer
      onClear={handleClear}
      placeholder="Search namespaces"
      loadingText="Searching namespaces..."
      isLoading={searchNamespace.isLoading}
      searchMode="allowTypeDuringSearch"
      onSearch={(query) =>
        searchNamespace.mutateAsync({
          query,
        })
      }
    />
  );
};
