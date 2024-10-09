import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import React, { useEffect, useState } from "react";
import { type ResponseStatus as Status, useLogSearchParams } from "../../query-state";

interface CheckboxItemProps {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}

const CheckboxItem = ({ id, label, description, checked, onCheckedChange }: CheckboxItemProps) => (
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
  const [checkedItems, setCheckedItems] = useState<Set<number>>(new Set());

  useEffect(() => {
    // Initialize checkedItems based on searchParams
    if (searchParams.responseStatutes) {
      setCheckedItems(new Set(searchParams.responseStatutes.map(Number)));
      setShowChecked(searchParams.responseStatutes.length > 0);
    }
  }, [searchParams.responseStatutes]);

  const handleItemChange = (status: number, checked: boolean) => {
    setCheckedItems((prev) => {
      const newSet = new Set(prev);
      if (checked) {
        newSet.add(Number(status));
      } else {
        newSet.delete(Number(status));
      }
      return newSet;
    });
  };

  const handleClear = () => {
    setCheckedItems(new Set());
    setShowChecked(false);
    setSearchParams((prevState) => ({
      ...prevState,
      responseStatutes: null,
    }));
  };

  const handleApply = () => {
    setShowChecked(true);
    setOpen(false);
    setSearchParams((prevState) => ({
      ...prevState,
      responseStatutes: Array.from(checkedItems) as Status[],
    }));
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div>
          Response Status {showChecked && checkedItems.size > 0 && `(${checkedItems.size})`}
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-80 bg-background p-0">
        {checkboxItems.map((item, index) => (
          <React.Fragment key={item.id}>
            <CheckboxItem
              {...item}
              checked={checkedItems.has(Number(item.id))}
              onCheckedChange={(checked) => handleItemChange(Number(item.id), checked)}
            />
            {index < checkboxItems.length - 1 && <div className="border-b border-border" />}
          </React.Fragment>
        ))}
        <div className="flex gap-2 p-2 w-full justify-end bg-background-subtle">
          <Button size="sm" variant="outline" onClick={handleClear}>
            Clear
          </Button>
          <Button size="sm" variant="primary" onClick={handleApply}>
            Apply
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
};
