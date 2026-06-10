"use client";

import { type VariantProps, cva } from "class-variance-authority";
// biome-ignore lint/style/useImportType: Biome wants this
import React from "react";
import { cn } from "../../lib/utils";

const fieldBaseClasses = "rounded-lg text-[13px] leading-5 transition-colors duration-300";

/**
 * The chrome of a text field: border, background, focus ring and text color.
 * The two maps below are the same strings apart from their focus prefix.
 * `fieldSurfaceClasses` uses `focus:`, so the element carrying it must be the
 * one that takes focus — a single `<input>`, `<textarea>` or trigger button.
 * `fieldGroupSurfaceClasses` uses `focus-within:`, for a container wrapping a
 * control and its addons.
 */
const fieldSurfaceClasses = {
  default:
    "border border-gray-5 hover:border-gray-8 bg-white dark:bg-gray-2 text-grayA-12 focus:border-accent-12 focus:ring-3 focus:ring-gray-5 focus:ring-offset-0 focus-visible:outline-hidden",
  ghost:
    "border border-transparent bg-transparent text-grayA-12 focus:border-accent-12 focus:ring-3 focus:ring-gray-5 focus:ring-offset-0 focus-visible:outline-hidden",
  success:
    "border border-success-9 hover:border-success-10 bg-white dark:bg-gray-2 text-success-11 focus:border-success-8 focus:ring-3 focus:ring-success-4 focus-visible:outline-hidden",
  warning:
    "border border-warning-9 hover:border-warning-10 bg-white dark:bg-gray-2 text-warning-11 focus:border-warning-8 focus:ring-3 focus:ring-warning-4 focus-visible:outline-hidden",
  error:
    "border border-error-9 hover:border-error-10 bg-white dark:bg-gray-2 text-error-11 focus:border-error-8 focus:ring-3 focus:ring-error-4 focus-visible:outline-hidden",
} as const;

const fieldInvalidClasses =
  "aria-invalid:border-error-9 aria-invalid:hover:border-error-10 aria-invalid:focus:border-error-8 aria-invalid:focus:ring-error-4";

const fieldGroupSurfaceClasses = {
  default:
    "border border-gray-5 hover:border-gray-8 bg-white dark:bg-gray-2 text-grayA-12 focus-within:border-accent-12 focus-within:ring-3 focus-within:ring-gray-5 focus-within:ring-offset-0 focus-visible:outline-hidden",
  ghost:
    "border border-transparent bg-transparent text-grayA-12 focus-within:border-accent-12 focus-within:ring-3 focus-within:ring-gray-5 focus-within:ring-offset-0 focus-visible:outline-hidden",
  success:
    "border border-success-9 hover:border-success-10 bg-white dark:bg-gray-2 text-success-11 focus-within:border-success-8 focus-within:ring-3 focus-within:ring-success-4 focus-visible:outline-hidden",
  warning:
    "border border-warning-9 hover:border-warning-10 bg-white dark:bg-gray-2 text-warning-11 focus-within:border-warning-8 focus-within:ring-3 focus-within:ring-warning-4 focus-visible:outline-hidden",
  error:
    "border border-error-9 hover:border-error-10 bg-white dark:bg-gray-2 text-error-11 focus-within:border-error-8 focus-within:ring-3 focus-within:ring-error-4 focus-visible:outline-hidden",
} as const;

const fieldGroupInvalidClasses =
  "has-[[aria-invalid=true]]:border-error-9 has-[[aria-invalid=true]]:hover:border-error-10 has-[[aria-invalid=true]]:focus-within:border-error-8 has-[[aria-invalid=true]]:focus-within:ring-error-4";

const inputGroupVariants = cva(
  [
    "flex w-full items-center",
    fieldBaseClasses,
    fieldGroupInvalidClasses,
    "has-[input:disabled]:opacity-50 has-[input:disabled]:cursor-not-allowed",
    "has-[textarea:disabled]:opacity-50 has-[textarea:disabled]:cursor-not-allowed",
  ],
  {
    variants: {
      variant: fieldGroupSurfaceClasses,
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
  return <div ref={ref} className={cn(inputGroupVariants({ variant }), className)} {...props} />;
}

type InputGroupInputProps = React.InputHTMLAttributes<HTMLInputElement> & {
  ref?: React.Ref<HTMLInputElement>;
};

function InputGroupInput({ className, ref, ...props }: InputGroupInputProps) {
  return (
    <input
      ref={ref}
      className={cn(
        "flex h-9 w-full min-w-0 flex-1 bg-transparent px-2 text-[13px] leading-5 text-grayA-12 placeholder:text-grayA-8 focus:outline-hidden disabled:cursor-not-allowed",
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
        "flex min-h-9 w-full min-w-0 flex-1 bg-transparent px-3 py-2 text-[13px] leading-5 text-grayA-12 placeholder:text-grayA-8 focus:outline-hidden disabled:cursor-not-allowed",
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
      className={cn("shrink-0 select-none text-[13px] leading-5 opacity-40", className)}
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
  fieldGroupSurfaceClasses,
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
