import { Button, InputGroup, InputGroupAddon, InputGroupInput } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { IconMagnifierOutline18, IconXmarkOutline18 } from "nucleo-ui-outline-18";
import { useEffect, useRef, useState } from "react";

// Generic filter type that can work with any filter structure
type BaseFilter = {
  field: string;
  id: string;
  operator: string;
  value: string | number;
};

type FilterHook<T extends BaseFilter = BaseFilter> = {
  filters: T[];
  updateFilters: (filters: T[]) => void;
  removeFilter: (id: string) => void;
};

type Props<T extends BaseFilter = BaseFilter> = {
  useFiltersHook: () => FilterHook<T>;
  placeholder?: string;
  debounceTime?: number;
  className?: string;
};

const MAX_QUERY_LENGTH = 120;
const DEFAULT_PLACEHOLDER = "Search...";
const DEFAULT_DEBOUNCE_MS = 300;

export const ListSearchInput = <T extends BaseFilter = BaseFilter>({
  useFiltersHook,
  placeholder = DEFAULT_PLACEHOLDER,
  debounceTime = DEFAULT_DEBOUNCE_MS,
  className,
}: Props<T>) => {
  const { filters, updateFilters } = useFiltersHook();
  const [searchText, setSearchText] = useState("");
  const [isInitialized, setIsInitialized] = useState(false);
  const debounceRef = useRef<NodeJS.Timeout | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const previousFilterValueRef = useRef<string>("");

  // Get current query filter value from URL on mount and when filters change
  useEffect(() => {
    const queryFilter = filters.find((f) => f.field === "query");
    const currentValue = typeof queryFilter?.value === "string" ? queryFilter.value : "";

    // Only update if the filter value actually changed (not from our own input)
    if (currentValue !== previousFilterValueRef.current) {
      previousFilterValueRef.current = currentValue;
      setSearchText(currentValue);
    }

    // Mark as initialized after first effect run
    if (!isInitialized) {
      setIsInitialized(true);
    }
  }, [filters, isInitialized]);

  // Cleanup debounce on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, []);

  const updateQuery = (value: string) => {
    // Remove existing filters for query field
    const filtersWithoutCurrent = filters.filter((f) => f.field !== "query");

    if (value.trim()) {
      // Reuse the existing query filter's id so repeated edits update one filter
      // in place instead of minting a fresh identity (and churning downstream
      // keys) on every change. Only generate an id when there is no query yet.
      const queryId = filters.find((f) => f.field === "query")?.id ?? crypto.randomUUID();
      updateFilters([
        ...filtersWithoutCurrent,
        {
          field: "query",
          id: queryId,
          operator: "contains",
          value: value.trim(),
        } as T,
      ]);
    } else {
      // Just remove query filters if empty
      updateFilters(filtersWithoutCurrent);
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setSearchText(value);

    // Debounce the filter update so we don't write URL state and refetch on
    // every keystroke. Enter, Escape, and clear bypass this for an immediate
    // update.
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }
    debounceRef.current = setTimeout(() => {
      updateQuery(value);
    }, debounceTime);
  };

  const handleClear = () => {
    setSearchText("");

    // Clear debounce
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }

    // Immediately update filters
    updateQuery("");
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Escape") {
      handleClear();
      inputRef.current?.blur();
    }

    if (e.key === "Enter") {
      // Clear debounce and immediately update
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
      updateQuery(searchText);
    }
  };

  // Show loading state while initializing
  if (!isInitialized) {
    return (
      <div className={cn("relative flex-1", className)}>
        <div
          className={cn(
            "px-2 flex items-center flex-1 gap-2 border border-solid border-gray-4 rounded-lg py-1 h-8 cursor-pointer",
            "bg-gray-3 opacity-50",
          )}
        >
          <div className="flex items-center gap-2 w-full flex-1">
            <div className="shrink-0">
              <IconMagnifierOutline18 className="text-accent-9 size-4" />
            </div>
            <div className="flex-1">
              <div className="text-accent-11 text-[13px] animate-pulse">Loading...</div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-center gap-2 w-full flex-1 h-8">
      <InputGroup
        variant="default"
        className={cn(
          "bg-transparent focus-within:ring-0 w-full h-8",
          "border border-solid border-gray-4 rounded-lg hover:bg-gray-3",
          "transition-all duration-200",
        )}
      >
        <InputGroupAddon className="pointer-events-none">
          <IconMagnifierOutline18 className="text-accent-9 size-4" />
        </InputGroupAddon>
        <InputGroupInput
          className="truncate text-accent-12 font-medium text-[13px] h-8 placeholder:text-accent-12 selection:bg-gray-6"
          ref={inputRef}
          type="text"
          value={searchText}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          maxLength={MAX_QUERY_LENGTH}
          placeholder={placeholder}
        />
        {searchText && (
          <InputGroupAddon align="inline-end">
            <Button
              variant="ghost"
              onClick={handleClear}
              className="text-accent-9 hover:text-accent-12 rounded-sm transition-colors shrink-0 cursor-pointer z-10"
              size="icon"
              aria-label="Clear search"
            >
              <IconXmarkOutline18 className="size-4 cursor-pointer" />
            </Button>
          </InputGroupAddon>
        )}
      </InputGroup>
    </div>
  );
};
