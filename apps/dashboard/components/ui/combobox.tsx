"use client";

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { Check, ChevronExpandY } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cva } from "class-variance-authority";
import * as React from "react";

const comboboxTriggerVariants = cva(
  "flex min-h-9 w-full rounded-lg text-[13px] leading-5 transition-colors duration-300 disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-grayA-8 text-grayA-12 items-center justify-between",
  {
    variants: {
      variant: {
        default: [
          "border border-gray-5 hover:border-gray-8 bg-gray-2 dark:bg-black",
          "focus:border focus:border-accent-12 focus:ring-2 focus:ring-gray-5 focus-visible:outline-none focus:ring-offset-0",
        ],
        success: [
          "border border-success-9 hover:border-success-10 bg-gray-2 dark:bg-black",
          "focus:border-success-8 focus:ring-2 focus:ring-success-2 focus-visible:outline-none",
        ],
        warning: [
          "border border-warning-9 hover:border-warning-10 bg-gray-2 dark:bg-black",
          "focus:border-warning-8 focus:ring-2 focus:ring-warning-2 focus-visible:outline-none",
        ],
        error: [
          "border border-error-9 hover:border-error-10 bg-gray-2 dark:bg-black",
          "focus:border-error-8 focus:ring-2 focus:ring-error-2 focus-visible:outline-none",
        ],
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

const comboboxWrapperVariants = cva("relative flex items-center w-full", {
  variants: {
    variant: {
      default: "text-grayA-12",
      success: "text-success-11",
      warning: "text-warning-11",
      error: "text-error-11",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

export type ComboboxOption = {
  label: React.ReactNode;
  value: string;
  searchValue?: string;
  selectedLabel?: React.ReactNode;
};

type ComboboxProps = {
  options: ComboboxOption[];
  value: string;
  onSelect: (value: string) => void;
  onChange?: (event: React.FormEvent<HTMLInputElement>) => void;
  placeholder?: React.ReactNode;
  searchPlaceholder?: string;
  emptyMessage?: React.ReactNode;
  disabled?: boolean;
  leftIcon?: React.ReactNode;
  wrapperClassName?: string;
  className?: string;
  variant?: "default" | "success" | "warning" | "error";
  id?: string;
  /** Additional accessibility attributes */
  "aria-describedby"?: string;
  "aria-invalid"?: boolean;
  "aria-required"?: boolean;
};

export function Combobox({
  options,
  value,
  onSelect,
  onChange,
  placeholder,
  searchPlaceholder = "Search...",
  emptyMessage = "No results found.",
  disabled = false,
  leftIcon,
  wrapperClassName,
  className,
  variant = "default",
  id,
  "aria-describedby": ariaDescribedby,
  "aria-invalid": ariaInvalid,
  "aria-required": ariaRequired,
  ...otherProps
}: ComboboxProps) {
  const [open, setOpen] = React.useState(false);

  const selectedOption = React.useMemo(
    () => options.find((option) => option.value === value),
    [options, value],
  );

  return (
    <Popover open={open} onOpenChange={setOpen} modal>
      <div className={cn(comboboxWrapperVariants({ variant }), wrapperClassName)}>
        {leftIcon && (
          <div className="absolute left-3 flex items-center pointer-events-none">{leftIcon}</div>
        )}
        <PopoverTrigger asChild className="w-full">
          <Button
            variant="outline"
            // biome-ignore lint/a11y/useSemanticElements: <explanation>
            role="combobox"
            aria-expanded={open}
            disabled={disabled}
            id={id}
            aria-describedby={ariaDescribedby}
            aria-invalid={ariaInvalid}
            aria-required={ariaRequired}
            className={cn(
              comboboxTriggerVariants({ variant }),
              "px-3 py-0",
              leftIcon && "pl-9",
              "pr-9", // Always have space for the chevron icon
              "h-auto justify-between font-normal w-full [&_svg]:size-3",
              className,
            )}
            {...otherProps}
          >
            {selectedOption ? (
              <div className="py-0 w-full">
                {selectedOption.selectedLabel || selectedOption.label}
              </div>
            ) : (
              placeholder
            )}
            <ChevronExpandY className="absolute right-3" iconSize="sm-regular" />
          </Button>
        </PopoverTrigger>
      </div>
      <PopoverContent className="p-0 w-full min-w-[var(--radix-popover-trigger-width)] rounded-lg border border-grayA-4 bg-white dark:bg-black shadow-md z-50">
        <Command>
          <CommandInput
            onInput={onChange}
            onKeyDown={(e) => {
              if (e.key !== "Enter" && e.key !== " ") {
                e.stopPropagation();
              }
            }}
            placeholder={searchPlaceholder}
            className="text-xs placeholder:text-xs placeholder:text-accent-8"
          />
          <CommandList className="max-h-[300px] overflow-y-auto overflow-x-hidden">
            <CommandEmpty>{emptyMessage}</CommandEmpty>
            <CommandGroup className="max-h-[260px] overflow-y-auto">
              {options.map((option) => (
                <CommandItem
                  key={option.value}
                  value={option.searchValue || option.value}
                  onSelect={() => {
                    onSelect(option.value);
                    setOpen(false);
                  }}
                  className="flex items-center py-0.5"
                >
                  {option.label}
                  <Check
                    className={cn("ml-auto", value === option.value ? "opacity-100" : "opacity-0")}
                    iconSize="sm-regular"
                  />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
