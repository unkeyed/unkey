import { useKeyboardShortcut } from "@/app/(app)/logs-v2/hooks/use-keyboard-shortcut";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { CaretRightOutline, CircleInfoSparkle, Magnifier, Refresh3 } from "@unkey/icons";
import { Button, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useRef, useState } from "react";

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
    <div className="group relative">
      <Button
        variant="ghost"
        className={cn(
          "group-data-[state=open]:bg-gray-4 px-2 group-data-[state=open]:ring-gray-12 group-data-[state=open]:ring-2 flex items-center gap-2 w-full",
          "focus-within:ring-2 focus-within:ring-gray-12 focus-within:bg-gray-4",
          "transition-all duration-200",
          searchText.length > 0 ? "bg-gray-4" : "",
          isLoading ? "bg-gray-4 ring-2 ring-accent-8" : "",
        )}
        disabled={isLoading}
      >
        <div className="flex items-center gap-2 relative">
          {isLoading ? (
            <div className="w-56">
              <Refresh3 className="text-accent-12 size-4 animate-spin" />
              <span className="text-accent-11 text-sm animate-pulse">Processing query...</span>
            </div>
          ) : (
            <>
              <Magnifier className="text-accent-9 size-4" />
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
            </>
          )}
        </div>
        <TooltipProvider>
          <Tooltip delayDuration={300}>
            <TooltipTrigger asChild>
              <div>
                <CircleInfoSparkle className="size-4 text-accent-9" />
              </div>
            </TooltipTrigger>
            <TooltipContent className="p-3 bg-gray-1 dark:bg-black drop-shadow-2xl border border-gray-6 rounded-lg text-accent-12 text-xs">
              <div>
                <div className="font-medium mb-2 flex items-center gap-2 text-[13px]">
                  <span>Try queries like:</span>
                  <span className="text-[11px] text-gray-11">(click to use)</span>
                </div>
                <ul className="space-y-1.5 pl-1 [&_svg]:size-[10px] ">
                  <li className="flex items-center gap-2">
                    <CaretRightOutline className="text-accent-9" />
                    <button
                      type="button"
                      className="hover:text-accent-11 transition-colors cursor-pointer hover:underline"
                      onClick={() => {
                        setSearchText("Show failed requests today");
                        handleSearch();
                      }}
                    >
                      "Show failed requests today"
                    </button>
                  </li>
                  <li className="flex items-center gap-2">
                    <CaretRightOutline className="text-accent-9" />
                    <button
                      type="button"
                      className="hover:text-accent-11 transition-colors cursor-pointer hover:underline"
                      onClick={() => {
                        setSearchText("auth errors in the last 3h");
                        handleSearch();
                      }}
                    >
                      "Auth errors in the last 3h"
                    </button>
                  </li>
                  <li className="flex items-center gap-2">
                    <CaretRightOutline className="size-2 text-accent-9" />
                    <button
                      type="button"
                      className="hover:text-accent-11 transition-colors cursor-pointer hover:underline"
                      onClick={() => {
                        setSearchText("API calls from a path that includes /api/v1/oz");
                        handleSearch();
                      }}
                    >
                      "API calls from a path that includes /api/v1/oz"
                    </button>
                  </li>
                </ul>
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </Button>
    </div>
  );
};
