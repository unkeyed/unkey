import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import { Button } from "@unkey/ui";
import { useCallback, useEffect } from "react";
import type { FilterOperator, FilterValue } from "../validation/filter.types";
import { useCheckboxState } from "./hooks";

export type BaseCheckboxOption = {
  id: number;
  checked: boolean;
  [key: string]: any;
};

type ExtractField<T> = T extends FilterValue<infer F, any, any> ? F : string;

type ExtractOperator<T> = T extends FilterValue<any, infer O, any> ? O : FilterOperator;

type ExtractValue<T> = T extends FilterValue<any, any, infer V> ? V : string | number;

interface BaseCheckboxFilterProps<
  TItem extends Record<string, any>,
  TFilterValue extends FilterValue<any, any, any>,
> {
  options: Array<{ id: number } & TItem>;
  filterField: ExtractField<TFilterValue>;
  checkPath: keyof TItem;
  className?: string;
  showScroll?: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement>;
  renderBottomGradient?: () => React.ReactNode;
  renderOptionContent?: (option: TItem) => React.ReactNode;
  createFilterValue: (option: TItem) => {
    value: ExtractValue<TFilterValue>;
    metadata?: TFilterValue["metadata"];
  };
  // Pass filter handling from outside
  filters: TFilterValue[];
  updateFilters: (newFilters: TFilterValue[]) => void;
  operator?: ExtractOperator<TFilterValue>;
  shouldSyncWithOptions?: boolean;
  // Selection mode
  selectionMode?: "multiple" | "single";
  // Allow deselection (clicking a selected item deselects it) - for single selection mode
  allowDeselection?: boolean;
  // Optional default selection index - for single selection mode
  defaultSelectionIndex?: number;
}

export const FilterCheckbox = <
  TItem extends Record<string, any>,
  TFilterValue extends FilterValue<any, any, any>,
>({
  options,
  filterField,
  checkPath,
  className,
  showScroll = false,
  renderOptionContent,
  createFilterValue,
  scrollContainerRef,
  renderBottomGradient,
  filters,
  updateFilters,
  operator = "is" as ExtractOperator<TFilterValue>,
  shouldSyncWithOptions = false,
  selectionMode = "multiple",
  allowDeselection = true,
  defaultSelectionIndex,
}: BaseCheckboxFilterProps<TItem, TFilterValue>) => {
  // Use the provided useCheckboxState hook
  const { checkboxes, handleCheckboxChange, handleSelectAll, handleKeyDown } = useCheckboxState<
    TItem,
    TFilterValue
  >({
    options,
    filters,
    filterField,
    checkPath,
    shouldSyncWithOptions,
  });

  // Handle single selection mode logic
  const handleSingleSelection = useCallback(
    (index: number) => {
      // If already checked and deselection is allowed, uncheck all
      if (checkboxes[index].checked && allowDeselection) {
        checkboxes.forEach((_, i) => {
          if (checkboxes[i].checked) {
            handleCheckboxChange(i);
          }
        });
        return;
      }

      // Uncheck all other checkboxes
      checkboxes.forEach((_, i) => {
        if (i !== index && checkboxes[i].checked) {
          handleCheckboxChange(i);
        }
      });

      // Check the selected checkbox if it's not already checked
      if (!checkboxes[index].checked) {
        handleCheckboxChange(index);
      }
    },
    [checkboxes, handleCheckboxChange, allowDeselection],
  );

  // Effect for default selection in single selection mode
  useEffect(() => {
    if (
      selectionMode === "single" &&
      defaultSelectionIndex !== undefined &&
      checkboxes.every((cb) => !cb.checked)
    ) {
      handleSingleSelection(defaultSelectionIndex);
    }
  }, [selectionMode, defaultSelectionIndex, checkboxes, handleSingleSelection]);

  // Handle checkbox click based on selection mode
  const handleCheckboxClick = useCallback(
    (index: number) => {
      if (selectionMode === "single") {
        handleSingleSelection(index);
      } else {
        handleCheckboxChange(index);
      }
    },
    [selectionMode, handleCheckboxChange, handleSingleSelection],
  );

  // Handle keyboard event
  const handleKeyboardEvent = useCallback(
    (event: React.KeyboardEvent<HTMLLabelElement>, index?: number) => {
      if (event.key === " " || event.key === "Enter") {
        event.preventDefault();
        if (index !== undefined) {
          handleCheckboxClick(index);
        } else if (selectionMode === "multiple") {
          handleSelectAll();
        }
      }

      // Use the handleKeyDown from the hook for other keyboard navigation
      handleKeyDown(event, index);
    },
    [handleCheckboxClick, handleSelectAll, handleKeyDown, selectionMode],
  );

  // Handle applying the filter
  const handleApplyFilter = useCallback(() => {
    const selectedCheckboxes = checkboxes.filter((c) => c.checked);
    const otherFilters = filters.filter((f) => f.field !== filterField);

    if (selectionMode === "single") {
      // For single selection, create at most one filter
      if (selectedCheckboxes.length > 0) {
        const filterValue = createFilterValue(selectedCheckboxes[0]);
        const newFilter = {
          id: crypto.randomUUID(),
          field: filterField,
          operator,
          ...filterValue,
        } as TFilterValue;

        updateFilters([...otherFilters, newFilter]);
      } else {
        // If nothing selected, just remove all filters for this field
        updateFilters(otherFilters);
      }
    } else {
      // For multiple selection, create a filter for each selected item
      const newFilters = selectedCheckboxes.map((checkbox) => {
        const filterValue = createFilterValue(checkbox);
        return {
          id: crypto.randomUUID(),
          field: filterField,
          operator,
          ...filterValue,
        } as TFilterValue;
      });

      updateFilters([...otherFilters, ...newFilters]);
    }
  }, [checkboxes, filterField, operator, filters, updateFilters, createFilterValue, selectionMode]);

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
        {selectionMode === "multiple" && (
          <div className="flex justify-between items-center">
            <label
              htmlFor="select-all-checkbox"
              className="flex items-center gap-[18px] cursor-pointer"
              // biome-ignore lint/a11y/useSemanticElements lint/a11y/noNoninteractiveElementToInteractiveRole: its okay
              role="checkbox"
              aria-checked={checkboxes.every((checkbox) => checkbox.checked)}
              onKeyDown={handleKeyboardEvent}
              tabIndex={0}
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
          </div>
        )}

        {checkboxes.map((checkbox, index) => (
          <label
            key={checkbox.id}
            htmlFor={`checkbox-${checkbox.id}`}
            className="flex gap-[18px] items-center py-1 cursor-pointer"
            // biome-ignore lint/a11y/useSemanticElements lint/a11y/noNoninteractiveElementToInteractiveRole: its okay
            role="checkbox"
            aria-checked={checkbox.checked}
            onKeyDown={(e) => handleKeyboardEvent(e, index)}
            tabIndex={0}
          >
            <Checkbox
              checked={checkbox.checked}
              className="size-4 rounded border-gray-4 [&_svg]:size-3"
              onClick={() => handleCheckboxClick(index)}
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
