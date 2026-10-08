"use client";

import { type VariantProps, cva } from "class-variance-authority";
// biome-ignore lint/style/useImportType: Biome wants this
import React from "react";
import { cn } from "../../lib/utils";
import {
  fieldBaseClasses,
  fieldFrameVariants,
  fieldInvalidClasses,
  fieldReadOnlyClasses,
  fieldReadOnlyFrameClasses,
  fieldSurfaceClasses,
} from "./input-group";

const inputVariants = cva(
  [
    fieldBaseClasses,
    "block w-full appearance-none px-[calc(--spacing(3.5)-1px)] py-[calc(--spacing(2.5)-1px)] sm:px-[calc(--spacing(3)-1px)] sm:py-[calc(--spacing(1.5)-1px)]",
    "text-base/6 sm:text-sm/6 placeholder:text-grayA-8 disabled:cursor-not-allowed",
    fieldInvalidClasses,
    fieldReadOnlyClasses,
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

// Hack to populate fumadocs' AutoTypeTable
type DocumentedInputProps = VariantProps<typeof inputVariants> & {
  wrapperClassName?: string;
};

type InputProps = DocumentedInputProps &
  React.InputHTMLAttributes<HTMLInputElement> & {
    ref?: React.Ref<HTMLInputElement>;
  };

function Input({ className, wrapperClassName, variant, ref, ...props }: InputProps) {
  return (
    <span
      data-slot="control"
      className={cn(fieldFrameVariants({ variant }), fieldReadOnlyFrameClasses, wrapperClassName)}
    >
      <input ref={ref} className={cn(inputVariants({ variant }), className)} {...props} />
    </span>
  );
}

export { Input, inputVariants, type InputProps, type DocumentedInputProps };
