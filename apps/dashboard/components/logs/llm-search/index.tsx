import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { cn } from "@/lib/utils";
import { CaretRightOutline, CircleInfoSparkle, Magnifier, Refresh3, XMark } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "components/ui/tooltip";
import { useEffect, useRef, useState } from "react";

type Props = {
  onSearch: (query: string) => void;
  onClear?: () => void;
  placeholder?: string;
  isLoading: boolean;
  hideExplainer?: boolean;
  hideClear?: boolean;
  loadingText?: string;
  clearingText?: string;
  searchOnChange?: boolean;
  debounceTime?: number;
};

export const LogsLLMSearch = ({
  onSearch,
  isLoading,
  onClear,
  hideExplainer = false,
  hideClear = false,
  placeholder = "Search and filter with AI…",
  loadingText = "AI consults the Palantír...",
  clearingText = "Clearing search...",
  searchOnChange = false,
  debounceTime = 500,
}: Props) => {
  const [searchText, setSearchText] = useState("");
  const [isClearingState, setIsClearingState] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceTimerRef = useRef<NodeJS.Timeout | null>(null);

  // Combined loading state that accounts for both search and clear operations
  const isClearing = isClearingState;
  const isProcessing = isLoading || isClearing;

  useKeyboardShortcut("s", () => {
    inputRef.current?.click();
    inputRef.current?.focus();
  });

  // Function to debounce clearing
  const debouncedClear = () => {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
    }

    setIsClearingState(true);

    debounceTimerRef.current = setTimeout(() => {
      onClear?.();
      setIsClearingState(false);
    }, debounceTime);
  };

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

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    const wasEmpty = searchText !== "";

    setSearchText(value);

    // If text was deleted completely, call onClear
    if (wasEmpty && value === "") {
      debouncedClear();
    }

    if (searchOnChange && value !== "") {
      // Clear any existing timer
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }

      // Set a new timer
      debounceTimerRef.current = setTimeout(() => {
        handleSearch(value);
      }, debounceTime);
    }
  };

  // Clean up the timer when the component unmounts
  useEffect(() => {
    return () => {
      if (debounceTimerRef.current) {
        clearTimeout(debounceTimerRef.current);
      }
    };
  }, []);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Escape") {
      e.preventDefault();

      // Always clear the input and debounce the onClear call
      setSearchText("");
      debouncedClear();

      // Blur the input by directly using the ref
      inputRef.current?.blur();
    }

    if (e.key === "Enter") {
      e.preventDefault();
      if (searchText !== "") {
        handleSearch(searchText);
      } else {
        debouncedClear();
      }
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
          isProcessing ? "bg-gray-4" : "",
        )}
      >
        <div className="flex items-center gap-2 w-80">
          <div className="flex-shrink-0">
            {isProcessing ? (
              <Refresh3 className="text-accent-10 size-4 animate-spin" />
            ) : (
              <Magnifier className="text-accent-9 size-4" />
            )}
          </div>

          <div className="flex-1">
            {isProcessing ? (
              <div className="text-accent-11 text-[13px] animate-pulse">
                {isLoading ? loadingText : clearingText}
              </div>
            ) : (
              <input
                ref={inputRef}
                type="text"
                value={searchText}
                onKeyDown={handleKeyDown}
                onChange={handleInputChange}
                placeholder={placeholder}
                className="text-accent-12 font-medium text-[13px] bg-transparent border-none outline-none focus:ring-0 focus:outline-none placeholder:text-accent-12 selection:bg-gray-6 w-full"
                disabled={isLoading}
              />
            )}
          </div>
        </div>

        {!isProcessing && (
          <>
            {searchText.length > 0 && !hideClear && (
              <button
                aria-label="Clear search"
                onClick={() => {
                  setSearchText("");
                  debouncedClear();
                }}
                type="button"
              >
                <XMark className="size-4 text-accent-9" />
              </button>
            )}
            {searchText.length === 0 && !hideExplainer && (
              <TooltipProvider>
                <Tooltip delayDuration={150}>
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
            )}
          </>
        )}
      </div>
    </div>
  );
};
