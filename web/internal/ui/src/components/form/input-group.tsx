"use client";

import { type VariantProps, cva } from "class-variance-authority";
// biome-ignore lint/style/useImportType: Biome wants this
import React from "react";
import { cn } from "../../lib/utils";

const fieldBaseClasses =
  "rounded-lg text-sm leading-5 transition-colors duration-300 focus:outline-hidden";

/**
 * A field is two layers. The frame wraps the control: its `::before` paints the
 * fill and drop shadow 1px inside the border, so the translucent border reads
 * against the page, and its `::after` draws the focus ring over the border. The
 * control itself carries only the border and text color.
 */
const fieldFrameVariants = cva(
  [
    "relative isolate block w-full",
    "before:absolute before:inset-px before:-z-10 before:rounded-[calc(var(--radius-lg)-1px)] before:bg-raised before:shadow-[0_1px_2px_0_rgb(0_0_0/0.07)] dark:before:hidden",
    "after:pointer-events-none after:absolute after:inset-0 after:rounded-lg after:ring-transparent after:ring-inset focus-within:after:ring-2",
    "has-disabled:opacity-50 has-disabled:before:bg-grayA-2 has-disabled:before:shadow-none",
    "has-aria-invalid:before:shadow-error-9/10 has-aria-invalid:focus-within:after:ring-error-8",
  ],
  {
    variants: {
      variant: {
        default: "focus-within:after:ring-grayA-9",
        ghost: "before:hidden focus-within:after:ring-grayA-9",
        success: "focus-within:after:ring-success-8",
        warning: "focus-within:after:ring-warning-8",
        error: "focus-within:after:ring-error-8",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

const fieldSurfaceClasses = {
  default:
    "border border-grayA-4 hover:border-grayA-7 bg-transparent dark:bg-grayA-2 text-grayA-12",
  ghost: "border border-transparent bg-transparent text-grayA-12",
  success:
    "border border-success-9 hover:border-success-10 bg-transparent dark:bg-grayA-2 text-success-11",
  warning:
    "border border-warning-9 hover:border-warning-10 bg-transparent dark:bg-grayA-2 text-warning-11",
  error: "border border-error-9 hover:border-error-10 bg-transparent dark:bg-grayA-2 text-error-11",
} as const;

const fieldInvalidClasses = "aria-invalid:border-error-9 aria-invalid:hover:border-error-10";

const fieldGroupInvalidClasses =
  "has-aria-invalid:border-error-9 has-aria-invalid:hover:border-error-10";

const inputGroupVariants = cva(
  [
    "flex h-9 w-full items-center has-[textarea]:h-auto has-disabled:cursor-not-allowed",
    "before:inset-0 after:-inset-px",
    fieldBaseClasses,
    fieldGroupInvalidClasses,
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

const inputGroupAddonVariants = cva("flex shrink-0 items-center gap-2", {
  variants: {
    align: {
      "inline-start": "pl-3",
      "inline-end": "pr-3",
    },
  },
  defaultVariants: {
    align: "inline-start",
  },
});

type DocumentedInputGroupProps = VariantProps<typeof inputGroupVariants>;

type InputGroupProps = DocumentedInputGroupProps &
  React.HTMLAttributes<HTMLDivElement> & {
    ref?: React.Ref<HTMLDivElement>;
  };

function InputGroup({ className, variant, ref, ...props }: InputGroupProps) {
  return (
    <div
      ref={ref}
      className={cn(fieldFrameVariants({ variant }), inputGroupVariants({ variant }), className)}
      {...props}
    />
  );
}

type InputGroupInputProps = React.InputHTMLAttributes<HTMLInputElement> & {
  ref?: React.Ref<HTMLInputElement>;
};

function InputGroupInput({ className, ref, ...props }: InputGroupInputProps) {
  return (
    <input
      ref={ref}
      className={cn(
        "flex h-full w-full min-w-0 flex-1 bg-transparent px-2 first:pl-[calc(--spacing(3)-1px)] text-sm leading-5 text-grayA-12 placeholder:text-grayA-8 focus:outline-hidden disabled:cursor-not-allowed",
        className,
      )}
      {...props}
    />
  );
}

type InputGroupTextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement> & {
  ref?: React.Ref<HTMLTextAreaElement>;
};

function InputGroupTextarea({ className, ref, ...props }: InputGroupTextareaProps) {
  return (
    <textarea
      ref={ref}
      className={cn(
        "flex min-h-9 w-full min-w-0 flex-1 bg-transparent px-3 py-2 text-sm leading-5 text-grayA-12 placeholder:text-grayA-8 focus:outline-hidden disabled:cursor-not-allowed",
        className,
      )}
      {...props}
    />
  );
}

type DocumentedInputGroupAddonProps = VariantProps<typeof inputGroupAddonVariants>;

type InputGroupAddonProps = DocumentedInputGroupAddonProps &
  React.HTMLAttributes<HTMLDivElement> & {
    ref?: React.Ref<HTMLDivElement>;
  };

function InputGroupAddon({ className, align, ref, ...props }: InputGroupAddonProps) {
  return <div ref={ref} className={cn(inputGroupAddonVariants({ align }), className)} {...props} />;
}

type InputGroupTextProps = React.HTMLAttributes<HTMLSpanElement> & {
  ref?: React.Ref<HTMLSpanElement>;
};

function InputGroupText({ className, ref, ...props }: InputGroupTextProps) {
  return (
    <span
      ref={ref}
      className={cn("shrink-0 select-none text-sm leading-5 opacity-40", className)}
      {...props}
    />
  );
}

export {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  InputGroupText,
  InputGroupTextarea,
  fieldBaseClasses,
  fieldFrameVariants,
  fieldInvalidClasses,
  fieldSurfaceClasses,
  type DocumentedInputGroupAddonProps,
  type DocumentedInputGroupProps,
  type InputGroupAddonProps,
  type InputGroupInputProps,
  type InputGroupProps,
  type InputGroupTextProps,
  type InputGroupTextareaProps,
};
