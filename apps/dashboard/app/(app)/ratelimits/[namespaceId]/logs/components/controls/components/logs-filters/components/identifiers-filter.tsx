import { Checkbox } from "@/components/ui/checkbox";
import { trpc } from "@/lib/trpc/client";
import { Button } from "@unkey/ui";
import { useCallback, useEffect, useRef, useState } from "react";
import { useRatelimitLogsContext } from "../../../../../context/logs";
import type { RatelimitFilterValue } from "../../../../../filters.schema";
import { useFilters } from "../../../../../hooks/use-filters";
import { useCheckboxState } from "./hooks/use-checkbox-state";

export const IdentifiersFilter = () => {
  const { namespaceId } = useRatelimitLogsContext();
  const {
    data: identifiers,
    isLoading,
    isError,
    refetch,
    isFetching,
  } = trpc.ratelimit.logs.queryDistinctIdentifiers.useQuery(
    { namespaceId },
    {
      select(identifiers) {
        return identifiers
          ? identifiers.map((identifier, index) => ({
              id: index + 1,
              path: identifier,
              checked: false,
            }))
          : [];
      },
    },
  );
  const { filters, updateFilters } = useFilters();
  const [isAtBottom, setIsAtBottom] = useState(false);
  const [needsScroll, setNeedsScroll] = useState(false);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const { checkboxes, handleCheckboxChange, handleSelectAll, handleKeyDown } = useCheckboxState({
    options: identifiers ?? [],
    filters,
    filterField: "identifiers",
    checkPath: "path",
    shouldSyncWithOptions: true,
  });

  useEffect(() => {
    const scrollContainer = scrollContainerRef.current;
    if (scrollContainer) {
      const checkScroll = () => {
        const { scrollTop, scrollHeight, clientHeight } = scrollContainer;
        const isBottom = Math.abs(scrollHeight - clientHeight - scrollTop) < 1;
        setIsAtBottom(isBottom);
        setNeedsScroll(scrollHeight > clientHeight);
      };

      scrollContainer.addEventListener("scroll", checkScroll);
      // Check initial state
      checkScroll();

      // Add resize observer to recheck on resize
      const resizeObserver = new ResizeObserver(checkScroll);
      resizeObserver.observe(scrollContainer);

      // Also check after a small delay to ensure content is rendered
      setTimeout(checkScroll, 100);

      return () => {
        scrollContainer.removeEventListener("scroll", checkScroll);
        resizeObserver.disconnect();
      };
    }
  }, []);

  const handleApplyFilter = useCallback(() => {
    const selectedPaths = checkboxes.filter((c) => c.checked).map((c) => c.path);

    // Keep all non-paths filters and add new path filters
    const otherFilters = filters.filter((f) => f.field !== "identifiers");
    const identifiersFilters: RatelimitFilterValue[] = selectedPaths.map((path) => ({
      id: crypto.randomUUID(),
      field: "identifiers",
      operator: "is",
      value: path,
    }));

    updateFilters([...otherFilters, ...identifiersFilters]);
  }, [checkboxes, filters, updateFilters]);

  if (isError) {
    return (
      <div className="flex flex-col bg-white rounded-lg shadow-sm min-w-[320px]">
        <div className="p-4 flex flex-col items-center gap-4">
          <div className="text-sm text-accent-12 max-h-64 overflow-auto text-center font-medium">
            Could not load identifiers
          </div>
        </div>
        <div className="border-t border-gray-4" />
        <div className="p-2">
          <Button
            variant="primary"
            className="font-sans w-full h-9 rounded-md"
            onClick={() => refetch()}
            loading={isFetching}
          >
            Try again
          </Button>
        </div>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center p-4">
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
        {needsScroll && !isAtBottom && (
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
