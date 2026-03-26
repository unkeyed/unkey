"use client";

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { cn } from "@/lib/utils";
import { Check, ChevronExpandY } from "@unkey/icons";
import { Button, Popover, PopoverContent, PopoverTrigger } from "@unkey/ui";
import { cva } from "class-variance-authority";
import * as React from "react";

const comboboxTriggerVariants = cva(
  "flex min-h-9 w-full rounded-lg text-[13px] leading-5 transition-colors duration-300 disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-grayA-8 text-grayA-12 items-center justify-between",
  {
    variants: {
      variant: {
        default: [
          "border border-gray-5 hover:border-gray-8 bg-gray-2 dark:bg-black",
          "focus:border focus:border-accent-12 focus:ring-2 focus:ring-gray-5 focus-visible:outline-hidden focus:ring-offset-0",
        ],
        success: [
          "border border-success-9 hover:border-success-10 bg-gray-2 dark:bg-black",
          "focus:border-success-8 focus:ring-2 focus:ring-success-2 focus-visible:outline-hidden",
        ],
        warning: [
          "border border-warning-9 hover:border-warning-10 bg-gray-2 dark:bg-black",
          "focus:border-warning-8 focus:ring-2 focus:ring-warning-2 focus-visible:outline-hidden",
        ],
        error: [
          "border border-error-9 hover:border-error-10 bg-gray-2 dark:bg-black",
          "focus:border-error-8 focus:ring-2 focus:ring-error-2 focus-visible:outline-hidden",
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
  /** When true the option is visible but not selectable and rendered at reduced opacity. */
  disabled?: boolean;
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
  /** Whether to close the popover on item select. Set to `false` for multi-select. */
  closeOnSelect?: boolean;
  /** Allow typing a custom value that isn't in the options list */
  creatable?: boolean;
  /** Hide the chevron icon in the trigger */
  hideChevron?: boolean;
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
  closeOnSelect = true,
  leftIcon,
  wrapperClassName,
  className,
  variant = "default",
  id,
  creatable = false,
  hideChevron = false,
  "aria-describedby": ariaDescribedby,
  "aria-invalid": ariaInvalid,
  "aria-required": ariaRequired,
  ...otherProps
}: ComboboxProps) {
  const [open, setOpen] = React.useState(false);
  const [search, setSearch] = React.useState("");

  const selectedOption = React.useMemo(
    () => options.find((option) => option.value === value),
    [options, value],
  );

  // When creatable and the current value isn't in options, inject it so cmdk shows it
  const effectiveOptions = React.useMemo(() => {
    if (creatable && value && !options.some((o) => o.value === value)) {
      return [{ label: value, value }, ...options];
    }
    return options;
  }, [creatable, value, options]);

  const showCreatableOption = React.useMemo(() => {
    if (!creatable || !search.trim()) {
      return false;
    }
    return !effectiveOptions.some((o) => o.value === search.trim());
  }, [creatable, search, effectiveOptions]);

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) {
          setSearch("");
        }
      }}
      modal={true}
    >
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
              !hideChevron && "pr-9", // Space for the chevron icon when visible
              "h-auto justify-between font-normal w-full [&_svg]:size-3",
              className,
            )}
            {...otherProps}
          >
            {selectedOption ? (
              <div className="py-0 w-full text-left">
                {selectedOption.selectedLabel || selectedOption.label}
              </div>
            ) : value && creatable ? (
              <div className="py-0 w-full text-left">{value}</div>
            ) : (
              <div className="text-left w-full">{placeholder}</div>
            )}
            {!hideChevron && <ChevronExpandY className="absolute right-3" iconSize="sm-regular" />}
          </Button>
        </PopoverTrigger>
      </div>
      <PopoverContent
        className="p-0 w-full min-w-(--radix-popover-trigger-width) rounded-lg border border-grayA-4 bg-white dark:bg-black shadow-md z-200 overflow-visible"
        onOpenAutoFocus={(e) => {
          // Let the CommandInput receive focus so users can type immediately
          e.preventDefault();
          if (e.currentTarget instanceof HTMLElement) {
            e.currentTarget.querySelector<HTMLInputElement>("[cmdk-input]")?.focus();
          }
        }}
      >
        <Command
          onKeyDown={(e) => {
            // Allow keyboard navigation within the combobox
            if (
              e.key === "ArrowDown" ||
              e.key === "ArrowUp" ||
              e.key === "Enter" ||
              e.key === "Escape"
            ) {
              e.stopPropagation();
            }
          }}
        >
          <CommandInput
            value={search}
            onValueChange={setSearch}
            onInput={onChange}
            onKeyDown={(e) => {
              // Prevent propagation to Dialog but allow command list navigation
              e.stopPropagation();
              // When creatable and Enter is pressed with no matching option, submit the typed value
              if (creatable && e.key === "Enter" && search.trim()) {
                const hasMatch = effectiveOptions.some(
                  (o) => (o.searchValue || o.value).toLowerCase() === search.trim().toLowerCase(),
                );
                if (!hasMatch) {
                  e.preventDefault();
                  onSelect(search.trim());
                  setSearch("");
                  if (closeOnSelect) {
                    setOpen(false);
                  }
                }
              }
            }}
            placeholder={searchPlaceholder}
            className="text-xs placeholder:text-xs placeholder:text-accent-8"
          />
          <CommandList className="max-h-[300px] overflow-y-auto overflow-x-hidden scrollbar-thin">
            {!showCreatableOption && <CommandEmpty>{emptyMessage}</CommandEmpty>}
            <CommandGroup className="max-h-[260px] overflow-y-auto">
              {showCreatableOption && (
                <CommandItem
                  value={search.trim()}
                  onSelect={() => {
                    onSelect(search.trim());
                    setSearch("");
                    if (closeOnSelect) {
                      setOpen(false);
                    }
                  }}
                  className="flex items-center py-1 mt-0 text-gray-9 text-xs"
                >
                  Use "{search.trim()}"
                </CommandItem>
              )}
              {effectiveOptions.map((option) => (
                <CommandItem
                  key={option.value}
                  value={option.searchValue || option.value}
                  disabled={option.disabled}
                  onSelect={() => {
                    if (option.disabled) {
                      return;
                    }
                    onSelect(option.value);
                    setSearch("");
                    if (closeOnSelect) {
                      setOpen(false);
                    }
                  }}
                  className={cn(
                    "flex items-center py-1 mt-0",
                    option.disabled && "opacity-50 cursor-not-allowed",
                  )}
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
