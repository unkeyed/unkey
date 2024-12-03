import { Checkbox } from "@/components/ui/checkbox";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import React, { useEffect, useState } from "react";
import {
  type ResponseStatus as Status,
  useLogSearchParams,
} from "../../../query-state";

interface CheckboxItemProps {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}

const checkboxItems = [
  { id: "5XX", label: "Error", description: "5XX error codes" },
  { id: "2XX", label: "Success", description: "2XX success codes" },
  { id: "4XX", label: "Warning", description: "4XX warning codes" },
];

export const ResponseStatus = () => {
  const [open, setOpen] = useState(false);
  const { searchParams, setSearchParams } = useLogSearchParams();
  const [checkedItems, setCheckedItems] = useState<Status[]>([]);

  useEffect(() => {
    if (searchParams.responseStatus) {
      setCheckedItems(searchParams.responseStatus);
    }
  }, [searchParams.responseStatus]);

  const handleItemChange = (status: Status, checked: boolean) => {
    const newCheckedItems = checked
      ? [...checkedItems, status]
      : checkedItems.filter((item) => item !== status);

    setCheckedItems(newCheckedItems);
    setSearchParams((prevState) => ({
      ...prevState,
      responseStatus: checkedItems,
    }));
  };

  const getStatusDisplay = () => {
    if (checkedItems.length === 0) {
      return "Response Status";
    }

    const statusLabels = checkedItems
      .map(
        (status) =>
          checkboxItems.find((item) => Number(item.id) === status)?.label
      )
      .filter(Boolean)
      .join(", ");

    return `Response Status (${statusLabels})`;
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div className="cursor-pointer">{getStatusDisplay()}</div>
      </PopoverTrigger>
      <PopoverContent className="w-80 bg-background p-0">
        {checkboxItems.map((item, index) => (
          <React.Fragment key={item.id}>
            <CheckboxItem
              {...item}
              checked={checkedItems.includes(Number(item.id) as Status)}
              onCheckedChange={(checked) => {
                handleItemChange(Number(item.id) as Status, checked);
              }}
            />
            {index < checkboxItems.length - 1 && (
              <div className="border-b border-border" />
            )}
          </React.Fragment>
        ))}
      </PopoverContent>
    </Popover>
  );
};

const CheckboxItem = ({
  id,
  label,
  description,
  checked,
  onCheckedChange,
}: CheckboxItemProps) => (
  <div className="items-top flex space-x-2 p-4">
    <Checkbox id={id} checked={checked} onCheckedChange={onCheckedChange} />
    <div className="grid gap-1.5 leading-none">
      <label
        htmlFor={id}
        className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
      >
        {label}
      </label>
      <p className="text-xs text-content-subtle">{description}</p>
    </div>
  </div>
);
