import { useCallback, useEffect, useState } from "react";
import type { FilterValue } from "@/app/(app)/logs-v2/filters.type";

type UseCheckboxStateProps<TItem> = {
  options: Array<{ id: number } & TItem>;
  filters: FilterValue[];
  filterField: string;
  valuePath: "value"; // Specify which field to get from filter
  checkPath: keyof TItem; // Specify which field to get from checkbox item
};

export const useCheckboxState = <TItem extends Record<string, any>>({
  options,
  filters,
  filterField,
  checkPath,
  valuePath,
}: UseCheckboxStateProps<TItem>) => {
  const [checkboxes, setCheckboxes] = useState(options);

  // Modified hook implementation
  useEffect(() => {
    const activeFilters = filters
      .filter((f) => f.field === filterField)
      .map((f) => f[valuePath]);

    setCheckboxes((prev) =>
      prev.map((checkbox) => ({
        ...checkbox,
        checked: activeFilters.includes(checkbox[checkPath]),
      }))
    );
  }, [filters, filterField, valuePath, checkPath]);

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

  const handleToggle = useCallback(
    (index?: number) => {
      if (typeof index === "number") {
        handleCheckboxChange(index);
      } else {
        handleSelectAll();
      }
    },
    [handleCheckboxChange, handleSelectAll]
  );

  const handleKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLLabelElement>, index?: number) => {
      // Handle checkbox toggle
      if (
        event.key === " " ||
        event.key === "Enter" ||
        event.key === "h" ||
        event.key === "l"
      ) {
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
        const currentIndex = Array.from(elements).findIndex(
          (el) => el === event.currentTarget
        );

        let nextIndex;
        if (event.key === "ArrowDown" || event.key === "j") {
          nextIndex = currentIndex < elements.length - 1 ? currentIndex + 1 : 0;
        } else {
          nextIndex = currentIndex > 0 ? currentIndex - 1 : elements.length - 1;
        }

        (elements[nextIndex] as HTMLElement).focus();
      }
    },
    [handleToggle]
  );

  return {
    checkboxes,
    handleCheckboxChange,
    handleSelectAll,
    handleKeyDown,
  };
};
