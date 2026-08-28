"use client";

import { type VariantProps, cva } from "class-variance-authority";
// biome-ignore lint/style/useImportType: Biome wants this
import React from "react";
import { cn } from "../../lib/utils";
import { fieldBaseClasses, fieldInvalidClasses, fieldSurfaceClasses } from "./input-group";

const inputVariants = cva(
  [
    "flex h-9 w-full px-2 py-2 disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-grayA-8",
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

// Hack to populate fumadocs' AutoTypeTable
type DocumentedInputProps = VariantProps<typeof inputVariants>;

type InputProps = DocumentedInputProps &
  React.InputHTMLAttributes<HTMLInputElement> & {
    ref?: React.Ref<HTMLInputElement>;
  };

function Input({ className, variant, ref, ...props }: InputProps) {
  return <input ref={ref} className={cn(inputVariants({ variant }), className)} {...props} />;
}

export { Input, inputVariants, type InputProps, type DocumentedInputProps };
