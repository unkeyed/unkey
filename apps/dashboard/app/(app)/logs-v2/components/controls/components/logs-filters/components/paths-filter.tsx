import type { FilterValue } from "@/app/(app)/logs-v2/filters.type";
import { useFilters } from "@/app/(app)/logs-v2/hooks/use-filters";
import { Checkbox } from "@/components/ui/checkbox";
import { trpc } from "@/lib/trpc/client";
import { Button } from "@unkey/ui";
import { useCallback, useEffect, useRef, useState } from "react";
import { useCheckboxState } from "./hooks/use-checkbox-state";

export const PathsFilter = () => {
  const { data: paths, isLoading } = trpc.logs.queryDistinctPaths.useQuery(undefined, {
    select(paths) {
      return paths
        ? paths.map((path, index) => ({
            id: index + 1,
            path,
            checked: false,
          }))
        : [];
    },
  });
  const { filters, updateFilters } = useFilters();
  const [isAtBottom, setIsAtBottom] = useState(false);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const { checkboxes, handleCheckboxChange, handleSelectAll, handleKeyDown } = useCheckboxState({
    options: paths ?? [],
    filters,
    filterField: "paths",
    checkPath: "path",
    shouldSyncWithOptions: true,
  });
  const handleScroll = useCallback(() => {
    if (scrollContainerRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current;
      const isBottom = Math.abs(scrollHeight - clientHeight - scrollTop) < 1;
      setIsAtBottom(isBottom);
    }
  }, []);

  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    if (scrollContainer) {
      scrollContainer.addEventListener("scroll", handleScroll);
      handleScroll();
      return () => {
        scrollContainer.removeEventListener("scroll", handleScroll);
      };
    }
  }, [handleScroll]);

  const handleApplyFilter = useCallback(() => {
    const selectedPaths = checkboxes.filter((c) => c.checked).map((c) => c.path);

    // Keep all non-paths filters and add new path filters
    const otherFilters = filters.filter((f) => f.field !== "paths");
    const pathFilters: FilterValue[] = selectedPaths.map((path) => ({
      id: crypto.randomUUID(),
      field: "paths",
      operator: "is",
      value: path,
    }));

    updateFilters([...otherFilters, ...pathFilters]);
  }, [checkboxes, filters, updateFilters]);

  if (isLoading) {
    return (
      <div
        className="flex flex-col items-center justify-center p-4"
        role="status"
        aria-live="polite"
      >
        <div className="flex items-center gap-3">
          <div className="h-4 w-4 animate-spin rounded-full border-2 border-accent-11 border-t-transparent" />
          <span className="text-sm text-accent-11">Loading paths...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col font-mono">
      <label
        className="flex items-center gap-4 px-4 pb-2 pt-4 cursor-pointer"
        // biome-ignore lint/a11y/noNoninteractiveElementToInteractiveRole: its okay
        role="checkbox"
        aria-checked={checkboxes.every((checkbox) => checkbox.checked)}
        onKeyDown={handleKeyDown}
      >
        <Checkbox
          checked={checkboxes.every((checkbox) => checkbox.checked)}
          className="size-4 rounded border-gray-4 [&_svg]:size-3"
          onClick={handleSelectAll}
        />
        <span className="text-xs text-accent-12">
          {checkboxes.every((checkbox) => checkbox.checked) ? "Unselect All" : "Select All"}
        </span>
      </label>
      <div className="relative px-2">
        <div
          ref={scrollContainerRef}
          className="flex flex-col gap-2 font-mono px-2 pb-2 max-h-64 overflow-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]"
        >
          {checkboxes.map((checkbox, index) => (
            <label
              key={checkbox.id}
              className="flex gap-[18px] items-center py-1 cursor-pointer"
              // biome-ignore lint/a11y/noNoninteractiveElementToInteractiveRole: its okay
              role="checkbox"
              aria-checked={checkbox.checked}
              onKeyDown={handleKeyDown}
            >
              <Checkbox
                checked={checkbox.checked}
                className="size-4 rounded border-gray-4 [&_svg]:size-3"
                onClick={() => handleCheckboxChange(index)}
              />
              <div className="text-accent-12 text-xs truncate">{checkbox.path}</div>
            </label>
          ))}
        </div>
        {!isAtBottom && (
          <div className="absolute bottom-0 left-0 right-0 h-12 pointer-events-none transition-opacity duration-200">
            <div className="h-full bg-gradient-to-t from-white to-white/0 dark:from-gray-900 dark:to-gray-900/0" />
          </div>
        )}
      </div>
      <div className="border-t border-gray-4" />
      <div className="p-2">
        <Button
          variant="primary"
          className="font-sans w-full h-9 rounded-md"
          onClick={handleApplyFilter}
        >
          Apply Filter
        </Button>
      </div>
    </div>
  );
};
