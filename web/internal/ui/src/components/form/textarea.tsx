import { cva, type VariantProps } from "class-variance-authority";
import type * as React from "react";
import { cn } from "../../lib/utils";
import { fieldBaseClasses, fieldInvalidClasses, fieldSurfaceClasses } from "./input-group";

const textareaVariants = cva(
  [
    "flex min-h-9 w-full px-3 py-2 disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-grayA-8",
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
type DocumentedTextareaProps = VariantProps<typeof textareaVariants>;

type TextareaProps = DocumentedTextareaProps &
  React.TextareaHTMLAttributes<HTMLTextAreaElement> & {
    ref?: React.Ref<HTMLTextAreaElement>;
  };

function Textarea({ className, variant, ref, ...props }: TextareaProps) {
  return <textarea ref={ref} className={cn(textareaVariants({ variant }), className)} {...props} />;
}

export { type DocumentedTextareaProps, Textarea, type TextareaProps, textareaVariants };
