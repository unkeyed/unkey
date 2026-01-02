import { useEffect, useState } from "react";
import type { FilterValue } from "../../validation/filter.types";

type UseCheckboxStateProps<TItem, TFilter extends FilterValue> = {
  options: Array<{ id: number } & TItem>;
  filters: TFilter[];
  filterField: string;
  checkPath: keyof TItem; // Specify which field to get from checkbox item
  shouldSyncWithOptions?: boolean;
};

export const useCheckboxState = <
  TItem extends Record<string, unknown>,
  TFilter extends FilterValue,
>({
  options,
  filters,
  filterField,
  checkPath,
  shouldSyncWithOptions = false,
}: UseCheckboxStateProps<TItem, TFilter>) => {
  const [checkboxes, setCheckboxes] = useState<TItem[]>(() => {
    const activeFilters = filters
      .filter((f) => f.field === filterField)
      .map((f) => String(f.value));
    return options.map((checkbox) => ({
      ...checkbox,
      checked: activeFilters.includes(String(checkbox[checkPath])),
    }));
  });

  useEffect(() => {
    if (shouldSyncWithOptions && options.length > 0) {
      setCheckboxes(options);
    }
  }, [options, shouldSyncWithOptions]);

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

  const handleKeyDown = (event: React.KeyboardEvent<HTMLLabelElement>, index?: number) => {
    // Special case for Escape key - let it bubble up naturally
    if (event.key === "Escape") {
      return;
    }

    // Handle checkbox toggle with Space or Enter
    if (event.key === " " || event.key === "Enter") {
      event.preventDefault();
      if (typeof index === "number") {
        handleCheckboxChange(index);
      } else {
        handleSelectAll();
      }
      return;
    }

    // Handle arrow navigation
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();

      // Get the parent container
      const container = event.currentTarget.closest(".flex-col");
      if (!container) {
        return;
      }

      // Get all labels using the 'for' attribute (not 'htmlFor' in the DOM)
      const checkboxLabels = container.querySelectorAll('label[for^="checkbox-"]');
      if (!checkboxLabels || checkboxLabels.length === 0) {
        return;
      }

      // Convert NodeList to Array for easier manipulation
      const labelsArray = Array.from(checkboxLabels);

      // Find the current element's index in the array
      const currentIndex = labelsArray.findIndex((label) => label === event.currentTarget);
      if (currentIndex === -1) {
        return;
      }

      // Calculate the next index based on arrow direction
      let nextIndex: number;
      if (event.key === "ArrowDown") {
        nextIndex = (currentIndex + 1) % labelsArray.length;
      } else {
        // ArrowUp
        nextIndex = (currentIndex - 1 + labelsArray.length) % labelsArray.length;
      }

      // Focus the next element
      try {
        (labelsArray[nextIndex] as HTMLElement).focus();
      } catch (e) {
        console.error("Focus error:", e);
      }
    }
  };

  return {
    checkboxes,
    handleCheckboxChange,
    handleSelectAll,
    handleKeyDown,
  };
};
