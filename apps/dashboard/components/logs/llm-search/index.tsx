import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { cn } from "@/lib/utils";
import { CaretRightOutline, CircleInfoSparkle, Magnifier, Refresh3, XMark } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "components/ui/tooltip";
import { useRef, useState } from "react";

type Props = {
  onSearch: (query: string) => void;
  isLoading: boolean;
};
export const LogsLLMSearch = ({ onSearch, isLoading }: Props) => {
  const [searchText, setSearchText] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useKeyboardShortcut("s", () => {
    inputRef.current?.click();
    inputRef.current?.focus();
  });

  const handleSearch = async (search: string) => {
    const query = search.trim();
    if (query) {
      try {
        onSearch(query);
      } catch (error) {
        console.error("Search failed:", error);
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Escape") {
      e.preventDefault();
      (document.activeElement as HTMLElement)?.blur();
    }
    if (e.key === "Enter") {
      e.preventDefault();
      handleSearch(searchText);
    }
  };

  const handlePresetQuery = (query: string) => {
    setSearchText(query);
    handleSearch(query);
  };

  return (
    <div className="group relative">
      <div
        className={cn(
          "group-data-[state=open]:bg-gray-4 px-2 flex items-center w-80 gap-2 border rounded-lg py-1 h-8 border-none cursor-pointer hover:bg-gray-3",
          "focus-within:bg-gray-4",
          "transition-all duration-200",
          searchText.length > 0 ? "bg-gray-4" : "",
          isLoading ? "bg-gray-4" : "",
        )}
      >
        <div className="flex items-center gap-2 w-80">
          <div className="flex-shrink-0">
            {isLoading ? (
              <Refresh3 className="text-accent-10 size-4 animate-spin" />
            ) : (
              <Magnifier className="text-accent-9 size-4" />
            )}
          </div>

          <div className="flex-1">
            {isLoading ? (
              <div className="text-accent-11 text-[13px] animate-pulse">
                AI consults the Palantír...
              </div>
            ) : (
              <input
                ref={inputRef}
                type="text"
                value={searchText}
                onKeyDown={handleKeyDown}
                onChange={(e) => setSearchText(e.target.value)}
                placeholder="Search and filter with AI…"
                className="text-accent-12 font-medium text-[13px] bg-transparent border-none outline-none focus:ring-0 focus:outline-none placeholder:text-accent-12 selection:bg-gray-6 w-full"
                disabled={isLoading}
              />
            )}
          </div>
        </div>{" "}
        <TooltipProvider>
          <Tooltip delayDuration={150}>
            {searchText.length > 0 && !isLoading && (
              <button aria-label="Clear search" onClick={() => setSearchText("")} type="button">
                <XMark className="size-4 text-accent-9" />
              </button>
            )}
            <TooltipTrigger asChild>
              {searchText.length === 0 && !isLoading && (
                <div>
                  <CircleInfoSparkle className="size-4 text-accent-9" />
                </div>
              )}
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
                      onClick={() => handlePresetQuery("Show failed requests today")}
                    >
                      "Show failed requests today"
                    </button>
                  </li>
                  <li className="flex items-center gap-2">
                    <CaretRightOutline className="text-accent-9" />
                    <button
                      type="button"
                      className="hover:text-accent-11 transition-colors cursor-pointer hover:underline"
                      onClick={() => handlePresetQuery("auth errors in the last 3h")}
                    >
                      "Auth errors in the last 3h"
                    </button>
                  </li>
                  <li className="flex items-center gap-2">
                    <CaretRightOutline className="size-2 text-accent-9" />
                    <button
                      type="button"
                      className="hover:text-accent-11 transition-colors cursor-pointer hover:underline"
                      onClick={() =>
                        handlePresetQuery("API calls from a path that includes /api/v1/oz")
                      }
                    >
                      "API calls from a path that includes /api/v1/oz"
                    </button>
                  </li>
                </ul>
              </div>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </div>
  );
};
