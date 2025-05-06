"use client";

import * as CheckboxPrimitive from "@radix-ui/react-checkbox";
import { Check } from "@unkey/icons";
import type { IconProps } from "@unkey/icons/src/props";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import { cn } from "../lib/utils";

const checkboxVariants = cva(
  "peer relative flex h-4 w-4 shrink-0 items-center justify-center rounded border transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed",
  {
    variants: {
      variant: {
        default: "",
        primary: [
          "border-grayA-6 data-[state=checked]:bg-accent-12 data-[state=checked]:border-transparent",
          "focus:ring-2 focus:ring-gray-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-grayA-4 disabled:data-[state=checked]:bg-grayA-6",
          "transition-all duration-200 ease-in-out",
        ],
        outline: [
          "border-grayA-6 bg-transparent data-[state=checked]:bg-transparent data-[state=checked]:border-grayA-8",
          "focus:border-grayA-12 focus:ring-2 focus:ring-gray-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-grayA-5 disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
        ghost: [
          "border-grayA-6 bg-transparent hover:bg-grayA-2 data-[state=checked]:bg-transparent data-[state=checked]:border-grayA-8",
          "focus:border-grayA-12 focus:ring-2 focus:ring-gray-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-grayA-4 disabled:opacity-70",
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
        md: "h-4 w-4",
        lg: "h-5 w-5",
        xlg: "h-6 w-6",
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
          "border-errorA-6 data-[state=checked]:bg-error-9 data-[state=checked]:border-transparent",
          "focus:border-error-11 focus:ring-2 focus:ring-error-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-errorA-4 disabled:data-[state=checked]:bg-error-6",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "outline",
        color: "danger",
        className: [
          "border-errorA-6 data-[state=checked]:border-error-9",
          "focus:border-error-11 focus:ring-2 focus:ring-error-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-errorA-4 disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "ghost",
        color: "danger",
        className: [
          "border-errorA-6 hover:bg-error-3 data-[state=checked]:border-error-9",
          "focus:border-error-11 focus:ring-2 focus:ring-error-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-errorA-4 disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      // Warning
      {
        variant: "primary",
        color: "warning",
        className: [
          "border-warningA-6 data-[state=checked]:bg-warning-8 data-[state=checked]:border-transparent",
          "focus:border-warning-11 focus:ring-2 focus:ring-warning-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-warningA-4 disabled:data-[state=checked]:bg-warning-6",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "outline",
        color: "warning",
        className: [
          "border-warningA-6 data-[state=checked]:border-warning-9",
          "focus:border-warning-11 focus:ring-2 focus:ring-warning-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-warningA-4 disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "ghost",
        color: "warning",
        className: [
          "border-warningA-6 hover:bg-warning-3 data-[state=checked]:border-warning-9",
          "focus:border-warning-11 focus:ring-2 focus:ring-warning-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-warningA-4 disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      // Success
      {
        variant: "primary",
        color: "success",
        className: [
          "border-successA-6 data-[state=checked]:bg-success-9 data-[state=checked]:border-transparent",
          "focus:border-success-11 focus:ring-2 focus:ring-success-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-successA-4 disabled:data-[state=checked]:bg-success-6",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "outline",
        color: "success",
        className: [
          "border-successA-6 data-[state=checked]:border-success-9",
          "focus:border-success-11 focus:ring-2 focus:ring-success-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-successA-4 disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "ghost",
        color: "success",
        className: [
          "border-successA-6 hover:bg-success-3 data-[state=checked]:border-success-9",
          "focus:border-success-11 focus:ring-2 focus:ring-success-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-successA-4 disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      // Info
      {
        variant: "primary",
        color: "info",
        className: [
          "border-infoA-6 data-[state=checked]:bg-info-9 data-[state=checked]:border-transparent",
          "focus:border-info-11 focus:ring-2 focus:ring-info-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-infoA-4 disabled:data-[state=checked]:bg-info-6",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "outline",
        color: "info",
        className: [
          "border-infoA-6 data-[state=checked]:border-info-9",
          "focus:border-info-11 focus:ring-2 focus:ring-info-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-infoA-4 disabled:opacity-70",
          "transition-all duration-200 ease-in-out",
        ],
      },
      {
        variant: "ghost",
        color: "info",
        className: [
          "border-infoA-6 hover:bg-info-3 data-[state=checked]:border-info-9",
          "focus:border-info-11 focus:ring-2 focus:ring-info-6 focus-visible:outline-none focus:ring-offset-0",
          "disabled:border-infoA-4 disabled:opacity-70",
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

const getIconSize = (size: CheckboxSize | undefined): IconProps["size"] => {
  switch (size) {
    case "sm":
      return "sm-regular";
    case "md":
      return "md-regular";
    case "lg":
      return "lg-regular";
    case "xlg":
      return "xl-regular";
    default:
      return "md-regular";
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
  checked?: boolean;

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

export type CheckboxProps = VariantProps<typeof checkboxVariants> &
  React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root> & {
    variant?: CheckboxVariant;
    color?: CheckboxColor;
    size?: CheckboxSize;
  };

const Checkbox = React.forwardRef<React.ElementRef<typeof CheckboxPrimitive.Root>, CheckboxProps>(
  ({ className, variant, color = "default", size, ...props }, ref) => {
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

    return (
      <CheckboxPrimitive.Root
        ref={ref}
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
          <Check size={iconSize} />
        </CheckboxPrimitive.Indicator>
      </CheckboxPrimitive.Root>
    );
  },
);

Checkbox.displayName = CheckboxPrimitive.Root.displayName;

export { Checkbox, checkboxVariants };
