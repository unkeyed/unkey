import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "../../lib/utils";
import {
  fieldBaseClasses,
  fieldFrameVariants,
  fieldInvalidClasses,
  fieldSurfaceClasses,
} from "./input-group";

const textareaVariants = cva(
  [
    fieldBaseClasses,
    "block min-h-9 w-full appearance-none px-[calc(--spacing(3.5)-1px)] py-[calc(--spacing(2.5)-1px)] sm:px-[calc(--spacing(3)-1px)] sm:py-[calc(--spacing(1.5)-1px)]",
    "text-base/6 sm:text-sm/6 placeholder:text-grayA-8 disabled:cursor-not-allowed",
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
type DocumentedTextareaProps = VariantProps<typeof textareaVariants> & {
  wrapperClassName?: string;
};

type TextareaProps = DocumentedTextareaProps &
  React.TextareaHTMLAttributes<HTMLTextAreaElement> & {
    ref?: React.Ref<HTMLTextAreaElement>;
  };

function Textarea({ className, wrapperClassName, variant, ref, ...props }: TextareaProps) {
  return (
    <span data-slot="control" className={cn(fieldFrameVariants({ variant }), wrapperClassName)}>
      <textarea ref={ref} className={cn(textareaVariants({ variant }), className)} {...props} />
    </span>
  );
}

export { Textarea, textareaVariants, type TextareaProps, type DocumentedTextareaProps };
