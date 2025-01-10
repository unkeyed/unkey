import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@unkey/ui";
import { CheckIcon } from "lucide-react";
import { useState } from "react";

interface CheckboxOption {
  id: number;
  status: string;
  label: string;
  color: string;
  checked: boolean;
}

const options: CheckboxOption[] = [
  {
    id: 1,
    status: "2xx",
    label: "Success",
    color: "bg-gray-7",
    checked: false,
  },
  {
    id: 2,
    status: "4xx",
    label: "Warning",
    color: "bg-warning-9",
    checked: false,
  },
  {
    id: 3,
    status: "4xx",
    label: "Error",
    color: "bg-error-9",
    checked: false,
  },
] as const;

export const StatusFilter = () => {
  const [checkboxes, setCheckboxes] = useState<CheckboxOption[]>(options);

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

  return (
    <div className="flex flex-col p-2">
      <div className="flex flex-col gap-2 font-mono px-2 py-2">
        <label className="flex items-center gap-2 cursor-pointer">
          <Checkbox
            id="select-all"
            checked={checkboxes.every((checkbox) => checkbox.checked)}
            className="size-[14px] rounded border-gray-4 [&_svg]:size-3"
            onClick={handleSelectAll}
          />
          <span className="text-xs text-accent-12 ml-2">Select All</span>
        </label>

        {checkboxes.map((checkbox, index) => (
          <label
            key={checkbox.id}
            className="flex gap-4 items-center py-1 cursor-pointer"
          >
            <Checkbox
              id={`checkbox-${checkbox.id}`}
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
        onClick={() => {
          const selectedItems = checkboxes.filter((c) => c.checked);
          console.info("Selected:", selectedItems);
        }}
      >
        Apply Filter
      </Button>
    </div>
  );
};
