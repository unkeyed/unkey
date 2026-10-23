"use client";

import { Select as SelectPrimitive } from "@base-ui/react/select";
import { IconCheckOutline12, IconChevronDownOutline18 } from "@unkey/icons";
import type * as React from "react";
import { type VariantProps, cva } from "../../lib/cva";
import { popupTransition } from "../../lib/popup";
import { cn } from "../../lib/utils";
import { fieldBaseClasses, fieldInvalidClasses, fieldSurfaceClasses } from "./input-group";

const selectTriggerVariants = cva(
  [
    "flex w-full items-center justify-between disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-grayA-8",
    fieldBaseClasses,
    fieldInvalidClasses,
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

type SelectSize = "md" | "sm";

const SELECT_SIZES: Record<SelectSize, { trigger: string; padding: string; chevron: string }> = {
  md: { trigger: "h-9", padding: "py-2 pr-9", chevron: "right-3 size-4" },
  sm: { trigger: "h-8 whitespace-nowrap", padding: "pr-8", chevron: "right-2 size-3.5" },
};

const selectWrapperVariants = cva("relative flex items-center w-full", {
  variants: {
    variant: {
      default: "text-grayA-12",
      ghost: "text-grayA-12",
      success: "text-success-11",
      warning: "text-warning-11",
      error: "text-error-11",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

// Types
export type DocumentedSelectProps = VariantProps<typeof selectTriggerVariants> & {
  size?: SelectSize;
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  wrapperClassName?: string;
};

const Select = SelectPrimitive.Root;
const SelectGroup = SelectPrimitive.Group;
function SelectValue({ className, ...props }: SelectPrimitive.Value.Props) {
  return (
    <SelectPrimitive.Value className={cn("flex min-w-0 items-center", className)} {...props} />
  );
}

function SelectTrigger({
  className,
  children,
  variant,
  size = "md",
  leftIcon,
  wrapperClassName,
  rightIcon,
  ref,
  ...props
}: SelectPrimitive.Trigger.Props &
  DocumentedSelectProps & {
    ref?: React.Ref<React.ComponentRef<typeof SelectPrimitive.Trigger>>;
  }) {
  return (
    <div className={cn(selectWrapperVariants({ variant }), wrapperClassName)}>
      {leftIcon && (
        <div className="absolute left-3 flex items-center pointer-events-none">{leftIcon}</div>
      )}
      <SelectPrimitive.Trigger
        ref={ref}
        className={cn(
          selectTriggerVariants({ variant, className: [SELECT_SIZES[size].trigger, className] }),
          "px-3",
          SELECT_SIZES[size].padding,
          leftIcon && "pl-9",
        )}
        {...props}
      >
        {children}
        <SelectPrimitive.Icon
          render={
            (rightIcon as React.ReactElement) || (
              <IconChevronDownOutline18
                className={cn("absolute text-gray-11", SELECT_SIZES[size].chevron)}
              />
            )
          }
        />
      </SelectPrimitive.Trigger>
    </div>
  );
}

function SelectContent({
  className,
  children,
  align,
  alignOffset,
  side,
  sideOffset = 4,
  // Radix parity: the old wrapper defaulted to position="popper" (popup
  // below the trigger). Base UI defaults to item-aligned (popup overlaps
  // the trigger and ignores side/sideOffset), so default it off.
  alignItemWithTrigger = false,
  maxRows,
  ref,
  ...props
}: SelectPrimitive.Popup.Props &
  Pick<
    SelectPrimitive.Positioner.Props,
    "align" | "alignOffset" | "side" | "sideOffset" | "alignItemWithTrigger"
  > & {
    /** Caps the list at this many rows of height. The rest scrolls. */
    maxRows?: number;
    ref?: React.Ref<React.ComponentRef<typeof SelectPrimitive.Popup>>;
  }) {
  return (
    <SelectPrimitive.Portal>
      <SelectPrimitive.Positioner
        className="isolate z-200"
        align={align}
        alignOffset={alignOffset}
        side={side}
        sideOffset={sideOffset}
        alignItemWithTrigger={alignItemWithTrigger}
      >
        <SelectPrimitive.Popup
          ref={ref}
          className={cn(
            "isolate relative overflow-hidden rounded-lg bg-raised text-gray-12 shadow-floating min-w-(--anchor-width) origin-(--transform-origin)",
            popupTransition,
            "data-[side=bottom]:data-starting-style:-translate-y-1 data-[side=top]:data-starting-style:translate-y-1 data-[side=left]:data-starting-style:translate-x-1 data-[side=right]:data-starting-style:-translate-x-1",
            className,
          )}
          {...props}
        >
          <SelectPrimitive.List
            className="overflow-y-auto overscroll-contain p-1"
            style={maxRows ? { maxHeight: `calc(${maxRows} * 2rem + 0.5rem)` } : undefined}
          >
            {children}
          </SelectPrimitive.List>
        </SelectPrimitive.Popup>
      </SelectPrimitive.Positioner>
    </SelectPrimitive.Portal>
  );
}

function SelectLabel({
  className,
  ref,
  ...props
}: SelectPrimitive.GroupLabel.Props & {
  ref?: React.Ref<React.ComponentRef<typeof SelectPrimitive.GroupLabel>>;
}) {
  return (
    <SelectPrimitive.GroupLabel
      ref={ref}
      className={cn("py-1.5 pl-2 pr-2 text-sm font-medium text-gray-11", className)}
      {...props}
    />
  );
}

function SelectItem({
  className,
  children,
  ref,
  ...props
}: SelectPrimitive.Item.Props & {
  ref?: React.Ref<React.ComponentRef<typeof SelectPrimitive.Item>>;
}) {
  return (
    <SelectPrimitive.Item
      ref={ref}
      className={cn(
        "relative flex w-full cursor-default select-none items-center rounded-md py-1.5 pl-2 pr-8 text-sm outline-hidden",
        "text-gray-12 hover:bg-gray-4 focus:bg-gray-5 data-highlighted:bg-gray-5 data-disabled:opacity-50",
        className,
      )}
      {...props}
    >
      <span className="absolute right-2 flex h-3.5 w-3.5 items-center justify-center">
        <SelectPrimitive.ItemIndicator>
          <IconCheckOutline12 className="text-gray-12" />
        </SelectPrimitive.ItemIndicator>
      </span>

      <SelectPrimitive.ItemText className="flex min-w-0 items-center">
        {children}
      </SelectPrimitive.ItemText>
    </SelectPrimitive.Item>
  );
}

function SelectSeparator({
  className,
  ref,
  ...props
}: SelectPrimitive.Separator.Props & {
  ref?: React.Ref<React.ComponentRef<typeof SelectPrimitive.Separator>>;
}) {
  return (
    <SelectPrimitive.Separator
      ref={ref}
      className={cn("-mx-1 my-1 h-px bg-gray-5", className)}
      {...props}
    />
  );
}

export {
  Select,
  SelectGroup,
  SelectValue,
  SelectTrigger,
  SelectContent,
  SelectLabel,
  SelectItem,
  SelectSeparator,
};
