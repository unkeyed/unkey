"use client";

import { Checkbox as CheckboxPrimitive } from "@base-ui/react/checkbox";
import { Check, Minus } from "@unkey/icons";
import type { IconProps } from "@unkey/icons/src/props";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import { cn } from "../../lib/utils";

const checkboxVariants = cva(
  "group peer relative flex h-4 w-4 shrink-0 items-center justify-center rounded-sm border transition-colors focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 data-disabled:cursor-not-allowed",
  {
    variants: {
      variant: {
        default: "",
        primary: [
          "border-grayA-6 data-checked:bg-accent-12 data-checked:border-transparent",
          "data-indeterminate:bg-accent-12 data-indeterminate:border-transparent",
          "focus:ring-3 focus:ring-gray-5 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-grayA-4 data-disabled:data-checked:bg-grayA-6",
          "transition-all duration-200 ease-in-out",
        ],
        outline: [
          "border-grayA-6 bg-transparent data-checked:bg-transparent data-checked:border-grayA-8",
          "data-indeterminate:bg-transparent data-indeterminate:border-grayA-8",
          "focus:border-grayA-12 focus:ring-3 focus:ring-gray-5 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-grayA-5 data-disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
        ghost: [
          "border-grayA-6 bg-transparent hover:bg-grayA-2 data-checked:bg-transparent data-checked:border-grayA-8",
          "data-indeterminate:bg-transparent data-indeterminate:border-grayA-8",
          "focus:border-grayA-12 focus:ring-3 focus:ring-gray-5 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-grayA-4 data-disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      color: {
        default: "",
        success: "",
        warning: "",
        danger: "",
        info: "",
      },
      size: {
        sm: "h-3 w-3",
        md: "h-3.5 w-3.5",
        lg: "h-4 w-4",
        xlg: "h-5 w-5",
      },
    },
    defaultVariants: {
      variant: "primary",
      color: "default",
      size: "md",
    },
    compoundVariants: [
      // Danger
      {
        variant: "primary",
        color: "danger",
        className: [
          "border-errorA-6 data-checked:bg-error-9 data-checked:border-transparent",
          "data-indeterminate:bg-error-9 data-indeterminate:border-transparent",
          "focus:border-error-11 focus:ring-3 focus:ring-error-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-errorA-4 data-disabled:data-checked:bg-error-6",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "outline",
        color: "danger",
        className: [
          "border-errorA-6 data-checked:border-error-9",
          "data-indeterminate:border-error-9",
          "focus:border-error-11 focus:ring-3 focus:ring-error-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-errorA-4 data-disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "ghost",
        color: "danger",
        className: [
          "border-errorA-6 hover:bg-error-3 data-checked:border-error-9",
          "data-indeterminate:border-error-9",
          "focus:border-error-11 focus:ring-3 focus:ring-error-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-errorA-4 data-disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      // Warning
      {
        variant: "primary",
        color: "warning",
        className: [
          "border-warningA-6 data-checked:bg-warning-8 data-checked:border-transparent",
          "data-indeterminate:bg-warning-8 data-indeterminate:border-transparent",
          "focus:border-warning-11 focus:ring-3 focus:ring-warning-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-warningA-4 data-disabled:data-checked:bg-warning-6",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "outline",
        color: "warning",
        className: [
          "border-warningA-6 data-checked:border-warning-9",
          "data-indeterminate:border-warning-9",
          "focus:border-warning-11 focus:ring-3 focus:ring-warning-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-warningA-4 data-disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "ghost",
        color: "warning",
        className: [
          "border-warningA-6 hover:bg-warning-3 data-checked:border-warning-9",
          "data-indeterminate:border-warning-9",
          "focus:border-warning-11 focus:ring-3 focus:ring-warning-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-warningA-4 data-disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      // Success
      {
        variant: "primary",
        color: "success",
        className: [
          "border-successA-6 data-checked:bg-success-9 data-checked:border-transparent",
          "data-indeterminate:bg-success-9 data-indeterminate:border-transparent",
          "focus:border-success-11 focus:ring-3 focus:ring-success-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-successA-4 data-disabled:data-checked:bg-success-6",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "outline",
        color: "success",
        className: [
          "border-successA-6 data-checked:border-success-9",
          "data-indeterminate:border-success-9",
          "focus:border-success-11 focus:ring-3 focus:ring-success-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-successA-4 data-disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "ghost",
        color: "success",
        className: [
          "border-successA-6 hover:bg-success-3 data-checked:border-success-9",
          "focus:border-success-11 focus:ring-3 focus:ring-success-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-successA-4 data-disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      // Info
      {
        variant: "primary",
        color: "info",
        className: [
          "border-infoA-6 data-checked:bg-info-9 data-checked:border-transparent",
          "focus:border-info-11 focus:ring-3 focus:ring-info-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-infoA-4 data-disabled:data-checked:bg-info-6",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "outline",
        color: "info",
        className: [
          "border-infoA-6 data-checked:border-info-9",
          "focus:border-info-11 focus:ring-3 focus:ring-info-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-infoA-4 data-disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "ghost",
        color: "info",
        className: [
          "border-infoA-6 hover:bg-info-3 data-checked:border-info-9",
          "focus:border-info-11 focus:ring-3 focus:ring-info-4 focus-visible:outline-hidden focus:ring-offset-0",
          "data-disabled:border-infoA-4 data-disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
    ],
  },
);

const checkmarkVariants = cva("flex items-center justify-center");

const VARIANT_MAP: Record<string, { variant: CheckboxVariant; color?: CheckboxColor }> = {
  default: { variant: "primary" },
  destructive: { variant: "primary", color: "danger" },
};

const getIconSize = (size: CheckboxSize | undefined): IconProps["iconSize"] => {
  switch (size) {
    case "sm":
      return "sm-regular";
    case "md":
      return "sm-regular";
    case "lg":
      return "md-regular";
    case "xlg":
      return "lg-regular";
    default:
      return "sm-regular";
  }
};

export type DocumentedCheckboxProps = VariantProps<typeof checkboxVariants> & {
  /**
   * The variant style to use for the checkbox
   * @default "primary"
   */
  variant?: CheckboxVariant;

  /**
   * The color scheme to use for the checkbox
   * @default "default"
   */
  color?: CheckboxColor;

  /**
   * The size of the checkbox
   * @default "md"
   */
  size?: CheckboxSize;

  /**
   * Whether the checkbox is checked
   */
  checked?: boolean | "indeterminate";

  /**
   * Default checked state when uncontrolled
   */
  defaultChecked?: boolean;

  /**
   * Whether the checkbox is disabled
   */
  disabled?: boolean;

  /**
   * Required state for the checkbox
   */
  required?: boolean;

  /**
   * Name of the checkbox for form submission
   */
  name?: string;

  /**
   * Value of the checkbox for form submission
   */
  value?: string;

  /**
   * Callback triggered when checkbox state changes
   */
  onCheckedChange?: (checked: boolean) => void;
};

type CheckboxVariant = NonNullable<VariantProps<typeof checkboxVariants>["variant"]>;
type CheckboxColor = NonNullable<VariantProps<typeof checkboxVariants>["color"]>;
type CheckboxSize = NonNullable<VariantProps<typeof checkboxVariants>["size"]>;

/**
 * Tri-state checked value. Base UI splits `checked` and `indeterminate`; this
 * wrapper keeps the ergonomic Radix-style tri-state API and translates it.
 */
export type CheckedState = boolean | "indeterminate";

export type CheckboxProps = VariantProps<typeof checkboxVariants> &
  Omit<
    React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>,
    "checked" | "indeterminate"
  > & {
    variant?: CheckboxVariant;
    color?: CheckboxColor;
    size?: CheckboxSize;
    /**
     * Whether the checkbox is checked. Accepts `"indeterminate"` for the mixed state.
     */
    checked?: CheckedState;
  };

const Checkbox = React.forwardRef<React.ComponentRef<typeof CheckboxPrimitive.Root>, CheckboxProps>(
  ({ className, variant, color = "default", size, checked, ...props }, ref) => {
    let mappedVariant: CheckboxVariant = "primary";
    let mappedColor: CheckboxColor = color;

    if (variant === null || variant === undefined) {
      mappedVariant = "primary";
    } else if (VARIANT_MAP[variant as keyof typeof VARIANT_MAP]) {
      const mapping = VARIANT_MAP[variant as keyof typeof VARIANT_MAP];
      mappedVariant = mapping.variant;
      if (mapping.color) {
        mappedColor = mapping.color;
      }
    } else {
      mappedVariant = variant as CheckboxVariant;
    }

    const iconSize = getIconSize(size);

    const checkmarkColor =
      mappedColor === "default" && mappedVariant === "primary"
        ? "text-white dark:text-black"
        : "text-white";

    const indeterminate = checked === "indeterminate";
    const checkedValue = checked === "indeterminate" ? false : checked;

    return (
      <CheckboxPrimitive.Root
        ref={ref}
        checked={checkedValue}
        indeterminate={indeterminate}
        className={cn(
          checkboxVariants({
            variant: mappedVariant,
            color: mappedColor,
            size,
            className,
          }),
        )}
        {...props}
      >
        <CheckboxPrimitive.Indicator className={cn(checkmarkVariants(), checkmarkColor)}>
          <Check iconSize={iconSize} className="hidden group-data-checked:block" />
          <Minus iconSize={iconSize} className="hidden group-data-indeterminate:block" />
        </CheckboxPrimitive.Indicator>
      </CheckboxPrimitive.Root>
    );
  },
);

Checkbox.displayName = "Checkbox";

export { Checkbox, checkboxVariants };
