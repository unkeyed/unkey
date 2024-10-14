import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import React, { useEffect, useState } from "react";
import {
  useLogSearchParams,
  type ResponseStatus as Status,
} from "../../../query-state";

interface CheckboxItemProps {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}

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

const checkboxItems = [
  { id: "500", label: "Error", description: "500 error codes" },
  { id: "200", label: "Success", description: "200 success codes" },
  { id: "400", label: "Warning", description: "400 success codes" },
];

export const ResponseStatus = () => {
  const [open, setOpen] = useState(false);
  const [showChecked, setShowChecked] = useState(false);
  const { searchParams, setSearchParams } = useLogSearchParams();
  const [checkedItem, setCheckedItem] = useState<Status | null>(null);

  useEffect(() => {
    if (searchParams.responseStatus) {
      setCheckedItem(searchParams.responseStatus);
      setShowChecked(!!searchParams.responseStatus);
    }
  }, [searchParams.responseStatus]);

  const handleItemChange = (status: Status, checked: boolean) => {
    if (checked) {
      setCheckedItem(status);
      setSearchParams((prevState) => ({
        ...prevState,
        responseStatus: status,
      }));
    } else {
      handleClear();
    }
  };

  const handleClear = () => {
    setCheckedItem(null);
    setShowChecked(false);
    setSearchParams((prevState) => ({
      ...prevState,
      responseStatus: null,
    }));
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div>
          Response Status {showChecked && checkedItem && `(${checkedItem})`}
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-80 bg-background p-0">
        {checkboxItems.map((item, index) => (
          <React.Fragment key={item.id}>
            <CheckboxItem
              {...item}
              checked={checkedItem === Number(item.id)}
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

export default ResponseStatus;
