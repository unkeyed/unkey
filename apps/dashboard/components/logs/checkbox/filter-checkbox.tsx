import type { LogsFilterValue } from "@/app/(app)/logs/filters.schema";
import { useFilters } from "@/app/(app)/logs/hooks/use-filters";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import { Button } from "@unkey/ui";
import { useCallback } from "react";
import type { FilterValue } from "../validation/filter.types";
import { useCheckboxState } from "./hooks";

export type BaseCheckboxOption = {
  id: number;
  checked: boolean;
  [key: string]: any;
};

interface BaseCheckboxFilterProps<
  TCheckbox extends BaseCheckboxOption,
  TFilter extends FilterValue,
> {
  options: TCheckbox[];
  filterField: "methods" | "paths" | "status";
  checkPath: string;
  className?: string;
  showScroll?: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement>;
  renderBottomGradient?: () => React.ReactNode;
  renderOptionContent?: (option: TCheckbox) => React.ReactNode;
  createFilterValue: (option: TCheckbox) => Pick<TFilter, "value" | "metadata">;
}

export const FilterCheckbox = <TCheckbox extends BaseCheckboxOption, TFilter extends FilterValue>({
  options,
  filterField,
  checkPath,
  className,
  showScroll = false,
  renderOptionContent,
  createFilterValue,
  scrollContainerRef,
  renderBottomGradient,
}: BaseCheckboxFilterProps<TCheckbox, TFilter>) => {
  const { filters, updateFilters } = useFilters();
  const { checkboxes, handleCheckboxChange, handleSelectAll, handleKeyDown } = useCheckboxState({
    options,
    filters,
    filterField,
    checkPath,
  });

  const handleApplyFilter = useCallback(() => {
    const selectedValues = checkboxes.filter((c) => c.checked).map((c) => createFilterValue(c));

    const otherFilters = filters.filter((f) => f.field !== filterField);
    const newFilters: LogsFilterValue[] = selectedValues.map((filterValue) => ({
      id: crypto.randomUUID(),
      field: filterField,
      operator: "is",
      ...filterValue,
    }));

    updateFilters([...otherFilters, ...newFilters]);
  }, [checkboxes, filterField, filters, updateFilters, createFilterValue]);

  return (
    <div className={cn("flex flex-col p-2", className)}>
      <div
        className={cn(
          "flex flex-col gap-2 font-mono px-2 py-2",
          showScroll &&
            "max-h-64 overflow-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]",
        )}
        ref={scrollContainerRef}
      >
        <div className="flex justify-between items-center">
          <label
            className="flex items-center gap-[18px] cursor-pointer"
            // biome-ignore lint/a11y/noNoninteractiveElementToInteractiveRole: its okay
            role="checkbox"
            aria-checked={checkboxes.every((checkbox) => checkbox.checked)}
            onKeyDown={handleKeyDown}
          >
            <Checkbox
              tabIndex={0}
              checked={checkboxes.every((checkbox) => checkbox.checked)}
              className="size-4 rounded border-gray-4 [&_svg]:size-3"
              onClick={handleSelectAll}
            />
            <span className="text-xs text-accent-12">
              {checkboxes.every((checkbox) => checkbox.checked) ? "Unselect All" : "Select All"}
            </span>
          </label>
        </div>
        {checkboxes.map((checkbox, index) => (
          <label
            key={checkbox.id}
            className="flex gap-[18px] items-center py-1 cursor-pointer"
            // biome-ignore lint/a11y/noNoninteractiveElementToInteractiveRole: its okay
            role="checkbox"
            aria-checked={checkbox.checked}
            onKeyDown={(e) => handleKeyDown(e, index)}
          >
            <Checkbox
              tabIndex={0}
              checked={checkbox.checked}
              className="size-4 rounded border-gray-4 [&_svg]:size-3"
              onClick={() => handleCheckboxChange(index)}
            />
            {renderOptionContent ? renderOptionContent(checkbox) : null}
          </label>
        ))}
      </div>

      {renderBottomGradient?.()}

      {renderBottomGradient && <div className="border-t border-gray-4" />}
      <Button
        variant="primary"
        className="font-sans mt-2 w-full h-9 rounded-md"
        onClick={handleApplyFilter}
      >
        Apply Filter
      </Button>
    </div>
  );
};
