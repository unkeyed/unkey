import { type FilterValue, type ResponseStatus, useFilters } from "@/app/(app)/logs-v2/query-state";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@unkey/ui";
import { useCallback, useEffect, useState } from "react";

interface CheckboxOption {
  id: number;
  status: ResponseStatus;
  label: string;
  color: string;
  checked: boolean;
}

const options: CheckboxOption[] = [
  {
    id: 1,
    status: 200,
    label: "Success",
    color: "bg-success-9",
    checked: false,
  },
  {
    id: 2,
    status: 400,
    label: "Warning",
    color: "bg-warning-8",
    checked: false,
  },
  {
    id: 3,
    status: 500,
    label: "Error",
    color: "bg-error-9",
    checked: false,
  },
];

export const StatusFilter = () => {
  const { filters, updateFilters } = useFilters();
  const [checkboxes, setCheckboxes] = useState<CheckboxOption[]>(options);

  // Sync checkboxes with filters on mount and when filters change
  useEffect(() => {
    const statusFilters = filters
      .filter((f) => f.field === "status")
      .map((f) => f.value as ResponseStatus);

    setCheckboxes((prev) =>
      prev.map((checkbox) => ({
        ...checkbox,
        checked: statusFilters.includes(checkbox.status),
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
    const selectedStatuses = checkboxes.filter((c) => c.checked).map((c) => c.status);

    // Keep all non-status filters and add new status filters
    const otherFilters = filters.filter((f) => f.field !== "status");
    const statusFilters: FilterValue[] = selectedStatuses.map((status) => ({
      id: crypto.randomUUID(),
      field: "status",
      operator: "is",
      value: status,
      metadata: {
        colorClass: status >= 500 ? "bg-error-9" : status >= 400 ? "bg-warning-8" : "bg-success-9",
      },
    }));

    updateFilters([...otherFilters, ...statusFilters]);
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
            <div className={`size-2 ${checkbox.color} rounded-[2px]`} />
            <span className="text-accent-9 text-xs">{checkbox.status}</span>
            <span className="text-accent-12 text-xs">{checkbox.label}</span>
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
