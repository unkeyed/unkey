import { InputSearch } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useState } from "react";
import { useFilters } from "../../../../../hooks/use-filters";

export const PathsFilter = () => {
  const { filters, updateFilters } = useFilters();
  const activeFilter = filters.find((f) => f.field === "paths");
  const [searchText, setSearchText] = useState(activeFilter?.value.toString() ?? "");
  const [isFocused, setIsFocused] = useState(false);

  const handleSearch = () => {
    const activeFilters = filters.filter((f) => f.field !== "paths");
    if (searchText.trim()) {
      updateFilters([
        ...activeFilters,
        {
          field: "paths",
          value: searchText,
          id: crypto.randomUUID(),
          operator: "contains",
        },
      ]);
    } else {
      updateFilters(activeFilters);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (isFocused) {
      e.stopPropagation();
      if (e.key === "Enter") {
        handleSearch();
      }
    }
  };

  const handleFocus = () => {
    setIsFocused(true);
  };

  const handleBlur = () => {
    setIsFocused(false);
  };

  return (
    <div className="flex flex-col p-4 gap-2 w-[300px]">
      <div className="relative w-full">
        <div
          className={cn(
            "flex items-center gap-2 px-2 py-1 h-8 rounded-md hover:bg-gray-3 bg-gray-4 transition-all duration-200",
            isFocused && "bg-gray-4",
          )}
        >
          <InputSearch className="w-4 h-4 text-accent-12" />
          <input
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            onKeyDown={handleKeyDown}
            onFocus={handleFocus}
            onBlur={handleBlur}
            type="text"
            placeholder="Search for path..."
            className="w-full text-[13px] font-medium text-accent-12 bg-transparent border-none outline-none focus:ring-0 focus:outline-none placeholder:text-accent-12"
          />
        </div>
      </div>
      <Button
        variant="primary"
        className="font-sans mt-2 w-full h-9 rounded-md"
        onClick={handleSearch}
      >
        Search
      </Button>
    </div>
  );
};
