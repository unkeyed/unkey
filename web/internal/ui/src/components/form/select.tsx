"use client";

import { Select as SelectPrimitive } from "@base-ui/react/select";
import { Check, ChevronDown } from "@unkey/icons";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import { cn } from "../../lib/utils";

const selectTriggerVariants = cva(
  "flex h-9 w-full rounded-lg text-[13px] leading-5 transition-colors duration-300 disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-grayA-8 text-grayA-12 items-center justify-between",
  {
    variants: {
      variant: {
        default: [
          "border border-gray-5 hover:border-gray-8 bg-gray-2 dark:bg-black",
          "focus:border focus:border-accent-12 focus:ring-3 focus:ring-gray-5 focus-visible:outline-hidden focus:ring-offset-0",
        ],
        success: [
          "border border-success-9 hover:border-success-10 bg-gray-2 dark:bg-black",
          "focus:border-success-8 focus:ring-3 focus:ring-success-4 focus-visible:outline-hidden",
        ],
        warning: [
          "border border-warning-9 hover:border-warning-10 bg-gray-2 dark:bg-black",
          "focus:border-warning-8 focus:ring-3 focus:ring-warning-4 focus-visible:outline-hidden",
        ],
        error: [
          "border border-error-9 hover:border-error-10 bg-gray-2 dark:bg-black",
          "focus:border-error-8 focus:ring-3 focus:ring-error-4 focus-visible:outline-hidden",
        ],
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

const selectWrapperVariants = cva("relative flex items-center w-full", {
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

// Types
export type DocumentedSelectProps = VariantProps<typeof selectTriggerVariants> & {
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  wrapperClassName?: string;
};

const Select = SelectPrimitive.Root;
const SelectGroup = SelectPrimitive.Group;
const SelectValue = SelectPrimitive.Value;

const SelectTrigger = React.forwardRef<
  React.ComponentRef<typeof SelectPrimitive.Trigger>,
  SelectPrimitive.Trigger.Props & DocumentedSelectProps
>(({ className, children, variant, leftIcon, wrapperClassName, rightIcon, ...props }, ref) => (
  <div className={cn(selectWrapperVariants({ variant }), wrapperClassName)}>
    {leftIcon && (
      <div className="absolute left-3 flex items-center pointer-events-none">{leftIcon}</div>
    )}
    <SelectPrimitive.Trigger
      ref={ref}
      className={cn(
        selectTriggerVariants({ variant, className }),
        "px-3 py-2",
        leftIcon && "pl-9",
        "pr-9", // Always have space for the chevron icon
      )}
      {...props}
    >
      {children}
      <SelectPrimitive.Icon
        render={
          (rightIcon as React.ReactElement) || (
            <ChevronDown className="absolute text-gray-11 right-3 w-4 h-4" iconSize="sm-medium" />
          )
        }
      />
    </SelectPrimitive.Trigger>
  </div>
));
SelectTrigger.displayName = "SelectTrigger";

const SelectContent = React.forwardRef<
  React.ComponentRef<typeof SelectPrimitive.Popup>,
  SelectPrimitive.Popup.Props &
    Pick<
      SelectPrimitive.Positioner.Props,
      "align" | "alignOffset" | "side" | "sideOffset" | "alignItemWithTrigger"
    >
>(
  (
    {
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
      ...props
    },
    ref,
  ) => (
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
            "isolate z-50 relative overflow-hidden rounded-lg border border-gray-5 bg-gray-2 text-gray-12 shadow-md min-w-(--anchor-width) origin-(--transform-origin)",
            "transition-[opacity,scale,translate] data-starting-style:opacity-0 data-starting-style:scale-95 data-ending-style:opacity-0 data-ending-style:scale-95",
            "data-[side=bottom]:data-starting-style:-translate-y-1 data-[side=top]:data-starting-style:translate-y-1 data-[side=left]:data-starting-style:translate-x-1 data-[side=right]:data-starting-style:-translate-x-1",
            className,
          )}
          {...props}
        >
          <SelectPrimitive.List className="p-1">{children}</SelectPrimitive.List>
        </SelectPrimitive.Popup>
      </SelectPrimitive.Positioner>
    </SelectPrimitive.Portal>
  ),
);
SelectContent.displayName = "SelectContent";

const SelectLabel = React.forwardRef<
  React.ComponentRef<typeof SelectPrimitive.GroupLabel>,
  SelectPrimitive.GroupLabel.Props
>(({ className, ...props }, ref) => (
  <SelectPrimitive.GroupLabel
    ref={ref}
    className={cn("py-1.5 pl-2 pr-2 text-[13px] font-medium text-gray-11", className)}
    {...props}
  />
));
SelectLabel.displayName = "SelectLabel";

const SelectItem = React.forwardRef<
  React.ComponentRef<typeof SelectPrimitive.Item>,
  SelectPrimitive.Item.Props
>(({ className, children, ...props }, ref) => (
  <SelectPrimitive.Item
    ref={ref}
    className={cn(
      "relative flex w-full cursor-default select-none items-center rounded-md py-1.5 pl-2 pr-8 text-[13px] outline-hidden",
      "text-gray-12 hover:bg-gray-4 focus:bg-gray-5 data-highlighted:bg-gray-5 data-disabled:opacity-50",
      className,
    )}
    {...props}
  >
    <span className="absolute right-2 flex h-3.5 w-3.5 items-center justify-center">
      <SelectPrimitive.ItemIndicator>
        <Check iconSize="sm-medium" className="text-gray-12" />
      </SelectPrimitive.ItemIndicator>
    </span>

    <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
  </SelectPrimitive.Item>
));
SelectItem.displayName = "SelectItem";

const SelectSeparator = React.forwardRef<
  React.ComponentRef<typeof SelectPrimitive.Separator>,
  SelectPrimitive.Separator.Props
>(({ className, ...props }, ref) => (
  <SelectPrimitive.Separator
    ref={ref}
    className={cn("-mx-1 my-1 h-px bg-gray-5", className)}
    {...props}
  />
));
SelectSeparator.displayName = "SelectSeparator";

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
