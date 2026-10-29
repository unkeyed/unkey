"use client";

import { IconChevronDownOutline18 } from "@unkey/icons";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import type { ReactNode } from "react";

export function ToolbarSelect<T extends string>({
  label,
  icon,
  value,
  options,
  onChange,
}: {
  label: string;
  icon: ReactNode;
  value: T;
  options: readonly { value: T; label: string }[];
  onChange: (value: T) => void;
}) {
  return (
    <Select
      value={value}
      items={options}
      onValueChange={(next) => {
        const option = options.find((candidate) => candidate.value === next);
        if (option) {
          onChange(option.value);
        }
      }}
    >
      <SelectTrigger
        aria-label={label}
        wrapperClassName="md:w-46"
        className="h-8 w-full bg-gray-1"
        leftIcon={icon}
        rightIcon={<IconChevronDownOutline18 className="size-3.5 absolute right-2" />}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
