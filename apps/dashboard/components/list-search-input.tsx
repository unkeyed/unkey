import { Magnifier } from "@unkey/icons";
import { Button, Input } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { Search, X } from "lucide-react";
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

export const ListSearchInput = <T extends BaseFilter = BaseFilter>({
  useFiltersHook,
  placeholder = DEFAULT_PLACEHOLDER,
  className,
}: Props<T>) => {
  const { filters, updateFilters } = useFiltersHook();
  const [searchText, setSearchText] = useState("");
  const [isInitialized, setIsInitialized] = useState(false);
  const debounceRef = useRef<NodeJS.Timeout>();
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
      // Add new filter
      updateFilters([
        ...filtersWithoutCurrent,
        {
          field: "query",
          id: crypto.randomUUID(),
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

    // Clear existing debounce
    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }

    updateQuery(value);
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
            "px-2 flex items-center flex-1 md:w-80 gap-2 border rounded-lg py-1 h-8 border-none cursor-pointer",
            "bg-gray-3 opacity-50",
          )}
        >
          <div className="flex items-center gap-2 w-full flex-1 md:w-80">
            <div className="flex-shrink-0">
              <Search className="text-accent-9 size-4" />
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
    <div className="flex items-center gap-2 w-full flex-1 md:w-80 h-8">
      <Input
        leftIcon={<Magnifier className="text-accent-9 size-4" />}
        className={cn(
          "truncate text-accent-12 font-medium text-[13px] bg-transparent border-none outline-none focus:ring-0 focus:outline-none placeholder:text-accent-12 selection:bg-gray-6 w-full h-8",
          "focus-within:bg-gray-4",
          "transition-all duration-200",
        )}
        rightIcon={
          searchText && (
            <Button
              variant="ghost"
              onClick={handleClear}
              className="text-accent-9 hover:text-accent-12 rounded transition-colors flex-shrink-0 cursor-pointer z-10"
              size="icon"
              aria-label="Clear search"
            >
              <X className="size-4 cursor-pointer" />
            </Button>
          )
        }
        ref={inputRef}
        variant="default"
        type="text"
        value={searchText}
        onChange={handleInputChange}
        onKeyDown={handleKeyDown}
        maxLength={MAX_QUERY_LENGTH}
        placeholder={placeholder}
      />
    </div>
  );
};
