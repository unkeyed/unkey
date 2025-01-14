import { type FilterValue, type HttpMethod, useFilters } from "@/app/(app)/logs-v2/query-state";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@unkey/ui";
import { useCallback, useEffect, useState } from "react";

interface CheckboxOption {
  id: number;
  method: HttpMethod;
  checked: boolean;
}

const options: CheckboxOption[] = [
  { id: 1, method: "GET", checked: false },
  { id: 2, method: "POST", checked: false },
  { id: 3, method: "PUT", checked: false },
  { id: 4, method: "DELETE", checked: false },
  { id: 5, method: "PATCH", checked: false },
] as const;

export const MethodsFilter = () => {
  const { filters, updateFilters } = useFilters();
  const [checkboxes, setCheckboxes] = useState<CheckboxOption[]>(options);

  // Sync checkboxes with filters on mount and when filters change
  useEffect(() => {
    const methodFilters = filters
      .filter((f) => f.field === "methods")
      .map((f) => f.value as HttpMethod);

    setCheckboxes((prev) =>
      prev.map((checkbox) => ({
        ...checkbox,
        checked: methodFilters.includes(checkbox.method),
      })),
    );
  }, [filters]);

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
    const selectedMethods = checkboxes.filter((c) => c.checked).map((c) => c.method);

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
          <label className="flex items-center gap-2 cursor-pointer">
            <Checkbox
              checked={checkboxes.every((checkbox) => checkbox.checked)}
              className="size-[14px] rounded border-gray-4 [&_svg]:size-3"
              onClick={handleSelectAll}
            />
            <span className="text-xs text-accent-12 ml-2">
              {checkboxes.every((checkbox) => checkbox.checked) ? "Unselect All" : "Select All"}
            </span>
          </label>
        </div>
        {checkboxes.map((checkbox, index) => (
          <label key={checkbox.id} className="flex gap-4 items-center py-1 cursor-pointer">
            <Checkbox
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
