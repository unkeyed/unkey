import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";

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
  { id: "error", label: "Error", description: "500 error codes" },
  { id: "success", label: "Success", description: "200 success codes" },
  { id: "warning", label: "Warning", description: "400 success codes" },
];

export const ResponseStatus = () => {
  const [open, setOpen] = useState(false);
  const [showChecked, setShowChecked] = useState(false);
  const [checkedItems, setCheckedItems] = useState<Set<string>>(new Set());

  const handleItemChange = (id: string, checked: boolean) => {
    setCheckedItems((prev) => {
      const newSet = new Set(prev);
      if (checked) {
        newSet.add(id);
      } else {
        newSet.delete(id);
      }
      return newSet;
    });
  };

  const handleClear = () => {
    setCheckedItems(new Set());
    setShowChecked(false);
  };

  const handleApply = () => {
    setShowChecked(true);
    setOpen(false);
    console.log("Applied filters:", Array.from(checkedItems));
    // Here you would typically update some parent component state or call an API
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <div>
          Response Status{" "}
          {showChecked && checkedItems.size > 0 && `(${checkedItems.size})`}
        </div>
      </PopoverTrigger>
      <PopoverContent className="w-80 bg-background p-0">
        {checkboxItems.map((item, index) => (
          <React.Fragment key={item.id}>
            <CheckboxItem
              {...item}
              checked={checkedItems.has(item.id)}
              onCheckedChange={(checked) => handleItemChange(item.id, checked)}
            />
            {index < checkboxItems.length - 1 && (
              <div className="border-b border-border" />
            )}
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
