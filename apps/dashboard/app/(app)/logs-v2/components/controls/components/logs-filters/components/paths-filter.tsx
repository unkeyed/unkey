import type { FilterValue } from "@/app/(app)/logs-v2/filters.type";
import { useFilters } from "@/app/(app)/logs-v2/hooks/use-filters";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@unkey/ui";
import { useCallback, useEffect, useRef, useState } from "react";

interface CheckboxOption {
  id: number;
  path: string;
  checked: boolean;
}

const options: CheckboxOption[] = [
  {
    id: 1,
    path: "/v1/analytics.export",
    checked: false,
  },
  {
    id: 2,
    path: "/v1/analytics.getDetails",
    checked: false,
  },
  {
    id: 3,
    path: "/v1/analytics.getOverview",
    checked: false,
  },
  {
    id: 4,
    path: "/v1/auth.login",
    checked: false,
  },
  {
    id: 5,
    path: "/v1/auth.logout",
    checked: false,
  },
  {
    id: 6,
    path: "/v1/auth.refreshToken",
    checked: false,
  },
  {
    id: 7,
    path: "/v1/data.delete",
    checked: false,
  },
  {
    id: 8,
    path: "/v1/data.fetch",
    checked: false,
  },
  {
    id: 9,
    path: "/v1/data.submit",
    checked: false,
  },
  {
    id: 10,
    path: "/v1/auth.login",
    checked: false,
  },
  {
    id: 11,
    path: "/v1/auth.logout",
    checked: false,
  },
  {
    id: 12,
    path: "/v1/auth.refreshToken",
    checked: false,
  },
  {
    id: 13,
    path: "/v1/data.delete",
    checked: false,
  },
  {
    id: 14,
    path: "/v1/data.fetch",
    checked: false,
  },
  {
    id: 15,
    path: "/v1/data.submit",
    checked: false,
  },
] as const;

export const PathsFilter = () => {
  const { filters, updateFilters } = useFilters();
  const [checkboxes, setCheckboxes] = useState<CheckboxOption[]>(options);
  const [isAtBottom, setIsAtBottom] = useState(false);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Sync checkboxes with filters on mount and when filters change
  useEffect(() => {
    const pathFilters = filters.filter((f) => f.field === "paths").map((f) => f.value as string);

    setCheckboxes((prev) =>
      prev.map((checkbox) => ({
        ...checkbox,
        checked: pathFilters.includes(checkbox.path),
      })),
    );
  }, [filters]);

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

  const handleCheckboxChange = (index: number): void => {
    setCheckboxes((prevCheckboxes) => {
      const newCheckboxes = [...prevCheckboxes];
      newCheckboxes[index] = {
        ...newCheckboxes[index],
        checked: !newCheckboxes[index].checked,
      };
      return newCheckboxes;
    });
  };

  const handleSelectAll = (): void => {
    setCheckboxes((prevCheckboxes) => {
      const allChecked = prevCheckboxes.every((checkbox) => checkbox.checked);
      return prevCheckboxes.map((checkbox) => ({
        ...checkbox,
        checked: !allChecked,
      }));
    });
  };

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

  return (
    <div className="flex flex-col font-mono">
      <label className="flex items-center gap-2 px-4 pb-2 pt-4 cursor-pointer">
        <Checkbox
          checked={checkboxes.every((checkbox) => checkbox.checked)}
          className="size-[14px] rounded border-gray-4 [&_svg]:size-3"
          onClick={handleSelectAll}
        />
        <span className="text-xs text-accent-12 ml-2">
          {checkboxes.every((checkbox) => checkbox.checked) ? "Unselect All" : "Select All"}
        </span>
      </label>
      <div className="relative px-2">
        <div
          ref={scrollContainerRef}
          className="flex flex-col gap-2 font-mono px-2 pb-2 max-h-64 overflow-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]"
        >
          {checkboxes.map((checkbox, index) => (
            <label key={checkbox.id} className="flex gap-4 items-center py-1 cursor-pointer">
              <Checkbox
                checked={checkbox.checked}
                className="size-[14px] rounded border-gray-4 [&_svg]:size-3"
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
