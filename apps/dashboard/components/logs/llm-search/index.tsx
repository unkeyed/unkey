import { useKeyboardShortcut } from "@/hooks/use-keyboard-shortcut";
import { cn } from "@/lib/utils";
import { useEffect, useRef, useState } from "react";
import { SearchActions } from "./components/search-actions";
import { SearchIcon } from "./components/search-icon";
import { SearchInput } from "./components/search-input";
import { useSearchStrategy } from "./hooks/use-search-strategy";

type SearchMode = "allowTypeDuringSearch" | "debounced" | "manual";

type Props = {
  onSearch: (query: string) => void;
  onClear?: () => void;
  placeholder?: string;
  isLoading: boolean;
  hideExplainer?: boolean;
  hideClear?: boolean;
  loadingText?: string;
  clearingText?: string;
  searchMode?: SearchMode;
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
  searchMode = "manual",
  debounceTime = 500,
}: Props) => {
  const [searchText, setSearchText] = useState("");
  const [isClearingState, setIsClearingState] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);

  const isClearing = isClearingState;
  const isProcessing = isLoading || isClearing;

  const { debouncedSearch, throttledSearch, executeSearch, clearDebounceTimer, resetSearchState } =
    useSearchStrategy(onSearch, debounceTime);
  useKeyboardShortcut("s", () => {
    inputRef.current?.click();
    inputRef.current?.focus();
  });

  const handleClear = () => {
    clearDebounceTimer();
    setIsClearingState(true);

    setTimeout(() => {
      onClear?.();
      setSearchText("");
    }, 0);

    setIsClearingState(false);
    resetSearchState();
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    const wasFilled = searchText !== "";

    setSearchText(value);

    // Handle clearing
    if (wasFilled && value === "") {
      handleClear();
      return;
    }

    // Skip if empty
    if (value === "") {
      return;
    }

    // Apply appropriate search strategy based on mode
    switch (searchMode) {
      case "allowTypeDuringSearch":
        throttledSearch(value);
        break;
      case "debounced":
        debouncedSearch(value);
        break;
      case "manual":
        // Do nothing - search triggered on Enter key or preset click
        break;
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Escape") {
      e.preventDefault();
      setSearchText("");
      handleClear();
      inputRef.current?.blur();
    }

    if (e.key === "Enter") {
      e.preventDefault();
      if (searchText !== "") {
        executeSearch(searchText);
      } else {
        handleClear();
      }
    }
  };

  const handlePresetQuery = (query: string) => {
    setSearchText(query);
    executeSearch(query);
  };

  // Clean up timers on unmount
  // biome-ignore lint/correctness/useExhaustiveDependencies: <explanation>
  useEffect(() => {
    return clearDebounceTimer();
  }, []);

  return (
    <div className="group relative" data-testid="logs-llm-search">
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
            <SearchIcon isProcessing={isProcessing} />
          </div>

          <div className="flex-1">
            <SearchInput
              value={searchText}
              placeholder={placeholder}
              isProcessing={isProcessing}
              isLoading={isLoading}
              loadingText={loadingText}
              clearingText={clearingText}
              searchMode={searchMode}
              onChange={handleInputChange}
              onKeyDown={handleKeyDown}
              inputRef={inputRef}
            />
          </div>
        </div>

        <SearchActions
          searchText={searchText}
          hideClear={hideClear}
          hideExplainer={hideExplainer}
          isProcessing={isProcessing}
          searchMode={searchMode}
          onClear={handleClear}
          onSelectExample={handlePresetQuery}
        />
      </div>
    </div>
  );
};
