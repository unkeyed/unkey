"use client";

import { Combobox as ComboboxPrimitive } from "@base-ui/react/combobox";
import { Check, ChevronExpandY, Magnifier } from "@unkey/icons";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import { cn } from "../../lib/utils";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  fieldBaseClasses,
  fieldInvalidClasses,
  fieldSurfaceClasses,
} from "./input-group";

const comboboxTriggerVariants = cva(
  [
    "flex h-9 w-full items-center justify-between gap-2 px-3 text-left font-normal",
    fieldBaseClasses,
    fieldInvalidClasses,
    "disabled:cursor-not-allowed disabled:opacity-50",
  ],
  {
    variants: {
      variant: fieldSurfaceClasses,
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

type DocumentedComboboxTriggerProps = VariantProps<typeof comboboxTriggerVariants>;

const ComboboxRoot = ComboboxPrimitive.Root;
const ComboboxValue = ComboboxPrimitive.Value;
const ComboboxCollection = ComboboxPrimitive.Collection;
const ComboboxGroup = ComboboxPrimitive.Group;
const ComboboxStatus = ComboboxPrimitive.Status;
const ComboboxPortal = ComboboxPrimitive.Portal;
const ComboboxPositioner = ComboboxPrimitive.Positioner;
const ComboboxPopup = ComboboxPrimitive.Popup;
const ComboboxRow = ComboboxPrimitive.Row;

type ComboboxTriggerProps = ComboboxPrimitive.Trigger.Props &
  DocumentedComboboxTriggerProps & {
    ref?: React.Ref<React.ComponentRef<typeof ComboboxPrimitive.Trigger>>;
  };

function ComboboxTrigger({ className, variant, ref, ...props }: ComboboxTriggerProps) {
  return (
    <ComboboxPrimitive.Trigger
      ref={ref}
      className={cn(comboboxTriggerVariants({ variant }), className)}
      {...props}
    />
  );
}

function ComboboxIcon({
  className,
  children,
  ref,
  ...props
}: ComboboxPrimitive.Icon.Props & {
  ref?: React.Ref<React.ComponentRef<typeof ComboboxPrimitive.Icon>>;
}) {
  return (
    <ComboboxPrimitive.Icon
      ref={ref}
      className={cn("flex shrink-0 items-center text-gray-11", className)}
      {...props}
    >
      {children ?? <ChevronExpandY iconSize="sm-regular" />}
    </ComboboxPrimitive.Icon>
  );
}

function ComboboxClear({
  className,
  ref,
  ...props
}: ComboboxPrimitive.Clear.Props & {
  ref?: React.Ref<React.ComponentRef<typeof ComboboxPrimitive.Clear>>;
}) {
  return (
    <ComboboxPrimitive.Clear
      ref={ref}
      className={cn("flex shrink-0 items-center text-gray-11 hover:text-gray-12", className)}
      {...props}
    />
  );
}

type ComboboxInputProps = ComboboxPrimitive.Input.Props & {
  icon?: React.ReactNode;
  wrapperClassName?: string;
  ref?: React.Ref<React.ComponentRef<typeof ComboboxPrimitive.Input>>;
};

function ComboboxInput({ className, icon, wrapperClassName, ref, ...props }: ComboboxInputProps) {
  return (
    <InputGroup className={cn("h-8", wrapperClassName)}>
      <InputGroupAddon className="text-gray-9">
        {icon ?? <Magnifier iconSize="sm-regular" />}
      </InputGroupAddon>
      <ComboboxPrimitive.Input
        ref={ref}
        className={cn("h-8 text-[13px] placeholder:text-grayA-8", className)}
        render={<InputGroupInput />}
        {...props}
      />
    </InputGroup>
  );
}

type ComboboxContentProps = ComboboxPrimitive.Popup.Props &
  Pick<ComboboxPrimitive.Positioner.Props, "align" | "alignOffset" | "side" | "sideOffset"> & {
    positionerClassName?: string;
    ref?: React.Ref<React.ComponentRef<typeof ComboboxPrimitive.Popup>>;
  };

function ComboboxContent({
  className,
  positionerClassName,
  align,
  alignOffset,
  side,
  sideOffset = 4,
  ref,
  ...props
}: ComboboxContentProps) {
  return (
    <ComboboxPrimitive.Portal>
      <ComboboxPrimitive.Positioner
        className={cn("isolate z-200", positionerClassName)}
        align={align}
        alignOffset={alignOffset}
        side={side}
        sideOffset={sideOffset}
      >
        <ComboboxPrimitive.Popup
          ref={ref}
          data-combobox-popup=""
          className={cn(
            "isolate relative z-50 flex flex-col overflow-hidden rounded-lg border border-gray-5 bg-background-overlay text-gray-12 shadow-md min-w-(--anchor-width) origin-(--transform-origin)",
            "transition-[opacity,scale,translate] data-starting-style:opacity-0 data-starting-style:scale-95 data-ending-style:opacity-0 data-ending-style:scale-95",
            "data-[side=bottom]:data-starting-style:-translate-y-1 data-[side=top]:data-starting-style:translate-y-1",
            className,
          )}
          {...props}
        />
      </ComboboxPrimitive.Positioner>
    </ComboboxPrimitive.Portal>
  );
}

function ComboboxList({
  className,
  ref,
  ...props
}: ComboboxPrimitive.List.Props & {
  ref?: React.Ref<React.ComponentRef<typeof ComboboxPrimitive.List>>;
}) {
  return (
    <ComboboxPrimitive.List
      ref={ref}
      className={cn("max-h-[300px] overflow-y-auto overflow-x-hidden p-1 empty:hidden", className)}
      {...props}
    />
  );
}

function ComboboxEmpty({
  className,
  ref,
  ...props
}: ComboboxPrimitive.Empty.Props & {
  ref?: React.Ref<React.ComponentRef<typeof ComboboxPrimitive.Empty>>;
}) {
  return (
    <ComboboxPrimitive.Empty
      ref={ref}
      className={cn("py-6 text-center text-[13px] text-grayA-9 empty:hidden", className)}
      {...props}
    />
  );
}

function ComboboxGroupLabel({
  className,
  ref,
  ...props
}: ComboboxPrimitive.GroupLabel.Props & {
  ref?: React.Ref<React.ComponentRef<typeof ComboboxPrimitive.GroupLabel>>;
}) {
  return (
    <ComboboxPrimitive.GroupLabel
      ref={ref}
      className={cn("px-2 py-1.5 text-xs font-medium text-grayA-9", className)}
      {...props}
    />
  );
}

function ComboboxItem({
  className,
  ref,
  ...props
}: ComboboxPrimitive.Item.Props & {
  ref?: React.Ref<React.ComponentRef<typeof ComboboxPrimitive.Item>>;
}) {
  return (
    <ComboboxPrimitive.Item
      ref={ref}
      className={cn(
        "relative flex w-full cursor-pointer select-none items-center overflow-hidden rounded-sm px-2 py-1.5 text-[13px] outline-hidden",
        "text-gray-12 data-highlighted:bg-grayA-3 data-disabled:cursor-not-allowed data-disabled:opacity-50",
        className,
      )}
      {...props}
    />
  );
}

function ComboboxItemIndicator({
  className,
  children,
  ref,
  ...props
}: ComboboxPrimitive.ItemIndicator.Props & {
  ref?: React.Ref<React.ComponentRef<typeof ComboboxPrimitive.ItemIndicator>>;
}) {
  return (
    <ComboboxPrimitive.ItemIndicator
      ref={ref}
      className={cn("ml-auto flex shrink-0 items-center", className)}
      {...props}
    >
      {children ?? <Check iconSize="sm-regular" />}
    </ComboboxPrimitive.ItemIndicator>
  );
}

export type ComboboxOption = {
  label: React.ReactNode;
  value: string;
  /** Text matched against the search query. Falls back to `value`. */
  searchValue?: string;
  /** Rendered in the trigger instead of `label` once selected. */
  selectedLabel?: React.ReactNode;
  /** When true the option is visible but not selectable and rendered at reduced opacity. */
  disabled?: boolean;
};

type DocumentedComboboxProps = DocumentedComboboxTriggerProps & {
  options: ComboboxOption[];
  value: string;
  onSelect: (value: string) => void;
  /** Fires on every keystroke in the search field. */
  onChange?: (event: React.FormEvent<HTMLInputElement>) => void;
  placeholder?: React.ReactNode;
  searchPlaceholder?: string;
  emptyMessage?: React.ReactNode;
  disabled?: boolean;
  /** Swaps the trigger's chevron for a spinner. */
  loading?: boolean;
  leftIcon?: React.ReactNode;
  wrapperClassName?: string;
  /** Offers the typed query as an option when it matches none of `options`. */
  creatable?: boolean;
  /** Class name applied to the popup container. */
  popoverClassName?: string;
};

type ComboboxProps = DocumentedComboboxProps &
  Pick<
    React.ButtonHTMLAttributes<HTMLButtonElement>,
    "className" | "id" | "title" | "aria-describedby" | "aria-invalid" | "aria-required"
  >;

function Combobox({
  options,
  value,
  onSelect,
  onChange,
  placeholder,
  searchPlaceholder = "Search...",
  emptyMessage = "No results found.",
  disabled = false,
  loading = false,
  leftIcon,
  wrapperClassName,
  className,
  variant = "default",
  creatable = false,
  popoverClassName,
  ...triggerProps
}: ComboboxProps) {
  const [open, setOpen] = React.useState(false);
  const [query, setQuery] = React.useState("");

  const trimmedQuery = query.trim();

  const effectiveOptions = React.useMemo(() => {
    if (creatable && value && !options.some((option) => option.value === value)) {
      return [{ label: value, value }, ...options];
    }
    return options;
  }, [creatable, value, options]);

  const createOption = React.useMemo((): ComboboxOption | undefined => {
    if (!creatable || !trimmedQuery) {
      return undefined;
    }
    const loweredQuery = trimmedQuery.toLowerCase();
    if (
      effectiveOptions.some(
        (option) => (option.searchValue || option.value).toLowerCase() === loweredQuery,
      )
    ) {
      return undefined;
    }
    return {
      label: <span className="truncate text-gray-9 text-xs">Use "{trimmedQuery}"</span>,
      value: trimmedQuery,
      searchValue: trimmedQuery,
    };
  }, [creatable, trimmedQuery, effectiveOptions]);

  const listOptions = React.useMemo(
    () => (createOption ? [createOption, ...effectiveOptions] : effectiveOptions),
    [createOption, effectiveOptions],
  );

  const optionsByValue = React.useMemo(
    () => new Map(listOptions.map((option) => [option.value, option])),
    [listOptions],
  );

  const itemValues = React.useMemo(() => listOptions.map((option) => option.value), [listOptions]);

  const selectedOption = effectiveOptions.find((option) => option.value === value);

  const filterOption = React.useCallback(
    (itemValue: string, itemQuery: string) => {
      const option = optionsByValue.get(itemValue);
      if (!option) {
        return false;
      }
      const haystack = option.searchValue || option.value;
      return haystack.toLowerCase().includes(itemQuery.trim().toLowerCase());
    },
    [optionsByValue],
  );

  return (
    <ComboboxRoot<string>
      items={itemValues}
      filter={filterOption}
      autoHighlight
      // Not `selectedOption?.value || null`: an option whose value is the empty
      // string still has to resolve to itself.
      value={selectedOption ? selectedOption.value : null}
      onValueChange={(next) => {
        if (next !== null) {
          onSelect(next);
        }
      }}
      inputValue={query}
      onInputValueChange={setQuery}
      open={open}
      onOpenChange={setOpen}
    >
      <div className={cn("relative flex w-full items-center", wrapperClassName)}>
        <ComboboxTrigger
          variant={variant}
          className={cn("[&_svg]:size-3", className)}
          disabled={disabled}
          {...triggerProps}
        >
          {leftIcon && <span className="flex shrink-0 items-center">{leftIcon}</span>}
          {selectedOption ? (
            <div className="w-full truncate text-left">
              {selectedOption.selectedLabel || selectedOption.label}
            </div>
          ) : value && creatable ? (
            <div className="w-full truncate text-left">{value}</div>
          ) : (
            <div className="w-full text-left">{placeholder}</div>
          )}
          {loading ? (
            <span className="size-3 shrink-0 animate-spin rounded-full border border-gray-6 border-t-gray-11" />
          ) : (
            <ComboboxIcon />
          )}
        </ComboboxTrigger>
      </div>
      <ComboboxContent className={popoverClassName}>
        <div className="p-1">
          <ComboboxInput placeholder={searchPlaceholder} onInput={onChange} />
        </div>
        <ComboboxEmpty>{emptyMessage}</ComboboxEmpty>
        <ComboboxList>
          {(itemValue: string) => {
            const option = optionsByValue.get(itemValue);
            if (!option) {
              return null;
            }
            return (
              <ComboboxItem key={option.value} value={option.value} disabled={option.disabled}>
                <span className="min-w-0 flex-1 truncate">{option.label}</span>
                <ComboboxItemIndicator />
              </ComboboxItem>
            );
          }}
        </ComboboxList>
      </ComboboxContent>
    </ComboboxRoot>
  );
}

export {
  Combobox,
  ComboboxClear,
  ComboboxCollection,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxGroup,
  ComboboxGroupLabel,
  ComboboxIcon,
  ComboboxInput,
  ComboboxItem,
  ComboboxItemIndicator,
  ComboboxList,
  ComboboxPopup,
  ComboboxPortal,
  ComboboxPositioner,
  ComboboxRoot,
  ComboboxRow,
  ComboboxStatus,
  ComboboxTrigger,
  ComboboxValue,
  comboboxTriggerVariants,
  type ComboboxContentProps,
  type ComboboxInputProps,
  type ComboboxProps,
  type ComboboxTriggerProps,
  type DocumentedComboboxProps,
  type DocumentedComboboxTriggerProps,
};
