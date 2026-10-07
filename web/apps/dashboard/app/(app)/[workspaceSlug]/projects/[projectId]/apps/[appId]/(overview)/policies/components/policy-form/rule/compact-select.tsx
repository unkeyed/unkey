"use client";

import { IconCheckOutline12, IconChevronDownOutline18 } from "@unkey/icons";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { cn } from "cn";

type Option<T> = { value: T; label: string };

export function CompactSelect<T extends string>({
  label,
  value,
  options,
  onChange,
  className,
}: {
  label: string;
  value: T;
  options: readonly Option<T>[];
  onChange: (value: T) => void;
  className?: string;
}) {
  return (
    <Select
      value={value}
      items={options}
      onValueChange={(next) => {
        const found = options.find((o) => o.value === next);
        if (found) {
          onChange(found.value);
        }
      }}
    >
      <SelectTrigger
        aria-label={label}
        wrapperClassName={className}
        className="h-8 whitespace-nowrap"
        rightIcon={<IconChevronDownOutline18 className="absolute right-2 size-3.5 text-gray-11" />}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent className="z-60">
        {options.map((o) => (
          <SelectItem key={o.value} value={o.value}>
            {o.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

export function CompactMultiSelect<T extends string>({
  label,
  values,
  options,
  onChange,
  placeholder,
  invalid,
  className,
}: {
  label: string;
  values: readonly T[];
  options: readonly Option<T>[];
  onChange: (values: T[]) => void;
  placeholder: string;
  invalid?: boolean;
  className?: string;
}) {
  const text = options
    .filter((o) => values.includes(o.value))
    .map((o) => o.label)
    .join(", ");
  return (
    <Popover>
      <PopoverTrigger
        aria-label={label}
        aria-invalid={invalid}
        className={cn(
          "relative flex h-8 items-center rounded-lg border border-grayA-5 bg-gray-1 pr-8 pl-3 text-left font-mono text-xs text-gray-12 transition-colors hover:border-grayA-7 aria-invalid:border-error-9 dark:bg-black",
          className,
        )}
      >
        <span className={cn("truncate", !text && "font-sans text-grayA-8")}>
          {text || placeholder}
        </span>
        <IconChevronDownOutline18 className="absolute right-2 size-3.5 text-gray-11" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-48 p-1">
        {options.map((o) => {
          const on = values.includes(o.value);
          return (
            <button
              key={o.value}
              type="button"
              aria-pressed={on}
              onClick={() =>
                onChange(on ? values.filter((v) => v !== o.value) : [...values, o.value])
              }
              className="flex h-8 w-full items-center justify-between rounded-md px-2 font-mono text-xs text-gray-12 hover:bg-grayA-3"
            >
              {o.label}
              {on ? <IconCheckOutline12 className="size-3" /> : null}
            </button>
          );
        })}
      </PopoverContent>
    </Popover>
  );
}
