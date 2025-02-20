import { LogsLLMSearch } from "@/components/logs/llm-search";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";

type LogsSearchProps = {
  setNamespaces: (namespaces: { id: string; name: string }[]) => void;
  initialNamespaces: { id: string; name: string }[];
};

export const LogsSearch = ({ setNamespaces, initialNamespaces }: LogsSearchProps) => {
  const searchNamespace = trpc.ratelimit.namespace.search.useMutation({
    onSuccess(data) {
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
    setNamespaces(initialNamespaces);
  };

  return (
    <LogsLLMSearch
      hideExplainer
      onClear={handleClear}
      placeholder="Search namespaces"
      isLoading={searchNamespace.isLoading}
      onSearch={(query) =>
        searchNamespace.mutateAsync({
          query,
        })
      }
    />
  );
};
