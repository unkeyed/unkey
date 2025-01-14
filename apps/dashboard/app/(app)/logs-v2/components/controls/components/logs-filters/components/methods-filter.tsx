import type { FilterValue, HttpMethod } from "@/app/(app)/logs-v2/filters.type";
import { useFilters } from "@/app/(app)/logs-v2/hooks/use-filters";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@unkey/ui";
import { useCallback } from "react";
import { useCheckboxState } from "./hooks/use-checkbox-state";

type CheckboxOption = {
  id: number;
  method: HttpMethod;
  checked: boolean;
};

const options: CheckboxOption[] = [
  { id: 1, method: "GET", checked: false },
  { id: 2, method: "POST", checked: false },
  { id: 3, method: "PUT", checked: false },
  { id: 4, method: "DELETE", checked: false },
  { id: 5, method: "PATCH", checked: false },
] as const;

export const MethodsFilter = () => {
  const { filters, updateFilters } = useFilters();
  const { checkboxes, handleCheckboxChange, handleSelectAll, handleKeyDown } =
    useCheckboxState({
      options,
      filters,
      filterField: "methods",
      valuePath: "value",
      checkPath: "method",
    });

  const handleApplyFilter = useCallback(() => {
    const selectedMethods = checkboxes
      .filter((c) => c.checked)
      .map((c) => c.method);

    // Keep all non-method filters and add new method filters
    const otherFilters = filters.filter((f) => f.field !== "methods");
    const methodFilters: FilterValue[] = selectedMethods.map((method) => ({
      id: crypto.randomUUID(),
      field: "methods",
      operator: "is",
      value: method,
    }));

    updateFilters([...otherFilters, ...methodFilters]);
  }, [checkboxes, filters, updateFilters]);

  return (
    <div className="flex flex-col p-2">
      <div className="flex flex-col gap-2 font-mono px-2 py-2">
        <div className="flex justify-between items-center">
          <label
            className="flex items-center gap-2 cursor-pointer"
            role="checkbox"
            aria-checked={checkboxes.every((checkbox) => checkbox.checked)}
            onKeyDown={handleKeyDown}
          >
            <Checkbox
              tabIndex={0}
              checked={checkboxes.every((checkbox) => checkbox.checked)}
              className="size-[14px] rounded border-gray-4 [&_svg]:size-3"
              onClick={handleSelectAll}
            />
            <span className="text-xs text-accent-12 ml-2">
              {checkboxes.every((checkbox) => checkbox.checked)
                ? "Unselect All"
                : "Select All"}
            </span>
          </label>
        </div>
        {checkboxes.map((checkbox, index) => (
          <label
            key={checkbox.id}
            className="flex gap-4 items-center py-1 cursor-pointer"
            role="checkbox"
            aria-checked={checkbox.checked}
            onKeyDown={(e) => handleKeyDown(e, index)}
          >
            <Checkbox
              tabIndex={0}
              checked={checkbox.checked}
              className="size-[14px] rounded border-gray-4 [&_svg]:size-3"
              onClick={() => handleCheckboxChange(index)}
            />
            <div className="text-accent-12 text-xs">{checkbox.method}</div>
          </label>
        ))}
      </div>
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
