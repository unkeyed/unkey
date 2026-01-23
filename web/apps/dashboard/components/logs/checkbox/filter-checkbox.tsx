import { cn } from "@/lib/utils";
import { Button, Checkbox } from "@unkey/ui";
import { useCallback, useEffect } from "react";
import type { FilterOperator, FilterValue } from "../validation/filter.types";
import { useCheckboxState } from "./hooks";

export type BaseCheckboxOption = {
  id: number;
  checked: boolean;
  // biome-ignore lint/suspicious/noExplicitAny: Generic object structure for flexible filter options
  [key: string]: any;
};

// biome-ignore lint/suspicious/noExplicitAny: Type extraction utilities need any for proper generic inference
type ExtractField<T> = T extends FilterValue<infer F, any, any> ? F : string;

// biome-ignore lint/suspicious/noExplicitAny: Type extraction utilities need any for proper generic inference
type ExtractOperator<T> = T extends FilterValue<any, infer O, any> ? O : FilterOperator;

// biome-ignore lint/suspicious/noExplicitAny: Type extraction utilities need any for proper generic inference
type ExtractValue<T> = T extends FilterValue<any, any, infer V> ? V : string | number;

interface BaseCheckboxFilterProps<
  // biome-ignore lint/suspicious/noExplicitAny: Generic item constraint requires any for flexible object access
  TItem extends Record<string, any>,
  // biome-ignore lint/suspicious/noExplicitAny: Generic filter constraint requires any for proper type inference
  TFilterValue extends FilterValue<any, any, any>,
> {
  options: Array<{ id: number } & TItem>;
  filterField: ExtractField<TFilterValue>;
  checkPath: keyof TItem;
  className?: string;
  showScroll?: boolean;
  scrollContainerRef?: React.RefObject<HTMLDivElement | null>;
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
  onDrawerClose?: () => void;
}

export const FilterCheckbox = <
  // biome-ignore lint/suspicious/noExplicitAny: Generic item constraint requires any for flexible object access
  TItem extends Record<string, any>,
  // biome-ignore lint/suspicious/noExplicitAny: Generic filter constraint requires any for proper type inference
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
  onDrawerClose,
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

    onDrawerClose?.();
  }, [
    checkboxes,
    filterField,
    operator,
    filters,
    updateFilters,
    createFilterValue,
    selectionMode,
    onDrawerClose,
  ]);

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
              // "checkbox-999 required to transfer focus from single checkboxes to this "select" all"
              htmlFor={"checkbox-999"}
              className="flex items-center gap-[18px] cursor-pointer"
              aria-checked={checkboxes.every((checkbox) => checkbox.checked)}
              onKeyDown={(e) => handleKeyDown(e)}
            >
              <Checkbox
                id={"checkbox-999"}
                checked={checkboxes.every((checkbox) => checkbox.checked)}
                className="size-4 rounded border-gray-4 [&_svg]:size-3"
                onClick={(e) => {
                  e.stopPropagation();
                  handleSelectAll();
                }}
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
            aria-checked={checkbox.checked}
            onKeyDown={(e) => handleKeyDown(e, index)}
          >
            <Checkbox
              id={`checkbox-${checkbox.id}`}
              checked={checkbox.checked}
              className="size-4 rounded border-gray-4 [&_svg]:size-3"
              onClick={(e) => {
                e.stopPropagation();
                handleCheckboxClick(index);
              }}
            />
            {renderOptionContent ? renderOptionContent(checkbox) : null}
          </label>
        ))}
      </div>

      {renderBottomGradient?.()}

      {renderBottomGradient && <div className="border-t border-gray-4" />}
      <Button
        variant="primary"
        className="mt-2 w-full h-9 rounded-md focus:ring-4 focus:ring-accent-9 focus:ring-offset-2"
        onClick={handleApplyFilter}
      >
        Apply Filter
      </Button>
    </div>
  );
};
