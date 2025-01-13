import { useKeyboardShortcut } from "@/app/(app)/logs-v2/hooks/use-keyboard-shortcut";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { Magnifier } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { Loader } from "lucide-react";
import { useRef, useState } from "react";
import { SearchPopover } from "./components/search-popover";

export const LogsSearch = () => {
  const queryLLMForStructuredQuery = trpc.logs.llmSearch.useMutation({
    onSuccess(data) {
      console.info("OUTPUT", data);
    },
    onError(error) {
      toast.error(error.message, {
        duration: 8000,
        important: true,
        position: "top-right",
        style: {
          whiteSpace: "pre-line",
        },
      });
    },
  });

  const [searchText, setSearchText] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useKeyboardShortcut("s", () => {
    inputRef.current?.click();
    inputRef.current?.focus();
  });

  const handleSearch = async () => {
    if (searchText.trim()) {
      try {
        await queryLLMForStructuredQuery.mutateAsync(searchText);
      } catch (error) {
        console.error("Search failed:", error);
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleSearch();
    }
  };

  const isLoading = queryLLMForStructuredQuery.isLoading;

  return (
    <SearchPopover>
      <div className="group">
        <Button
          variant="ghost"
          className={cn(
            "group-data-[state=open]:bg-gray-4 px-2 group-data-[state=open]:ring-gray-12 group-data-[state=open]:ring-2 flex items-center gap-2",
            "focus-within:ring-2 focus-within:ring-gray-12 focus-within:bg-gray-4",
            {
              "opacity-100": !isLoading,
              "opacity-0": isLoading,
            },
            searchText.length > 0 ? "ring-2 ring-gray-12 bg-gray-4" : "",
          )}
          aria-label="Search logs"
          aria-haspopup="true"
          title="Press 'S' to toggle filters"
          disabled={isLoading}
        >
          {isLoading ? (
            <Loader className="text-accent-9 size-4 animate-spin" />
          ) : (
            <Magnifier className="text-accent-9 size-4" />
          )}
          <input
            ref={inputRef}
            type="text"
            value={searchText}
            onKeyDown={handleKeyDown}
            onChange={(e) => setSearchText(e.target.value)}
            placeholder="Search and filter with AI…"
            className="text-accent-12 font-medium text-[13px] bg-transparent border-none outline-none focus:ring-0 focus:outline-none placeholder:text-accent-12 selection:bg-gray-6"
            disabled={isLoading}
          />
        </Button>
      </div>
    </SearchPopover>
  );
};
