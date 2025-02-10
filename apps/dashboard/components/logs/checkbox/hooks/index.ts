import { useEffect, useState } from "react";
import type { FilterValue } from "../../validation/filter.types";

type UseCheckboxStateProps<TItem, TFilter extends FilterValue> = {
  options: Array<{ id: number } & TItem>;
  filters: TFilter[];
  filterField: string;
  checkPath: keyof TItem; // Specify which field to get from checkbox item
  shouldSyncWithOptions?: boolean;
};

export const useCheckboxState = <TItem extends Record<string, any>, TFilter extends FilterValue>({
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

  const handleToggle = (index?: number) => {
    if (typeof index === "number") {
      handleCheckboxChange(index);
    } else {
      handleSelectAll();
    }
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLLabelElement>, index?: number) => {
    // Handle checkbox toggle
    if (event.key === " " || event.key === "Enter" || event.key === "h" || event.key === "l") {
      event.preventDefault();
      handleToggle(index);
    }

    // Handle navigation
    if (
      event.key === "ArrowDown" ||
      event.key === "ArrowUp" ||
      event.key === "j" ||
      event.key === "k"
    ) {
      event.preventDefault();
      const elements = document.querySelectorAll('label[role="checkbox"]');
      const currentIndex = Array.from(elements).findIndex((el) => el === event.currentTarget);

      let nextIndex: number;
      if (event.key === "ArrowDown" || event.key === "j") {
        nextIndex = currentIndex < elements.length - 1 ? currentIndex + 1 : 0;
      } else {
        nextIndex = currentIndex > 0 ? currentIndex - 1 : elements.length - 1;
      }

      (elements[nextIndex] as HTMLElement).focus();
    }
  };

  return {
    checkboxes,
    handleCheckboxChange,
    handleSelectAll,
    handleKeyDown,
  };
};
