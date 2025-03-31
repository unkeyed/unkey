import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import { cn } from "../lib/utils";

const textareaVariants = cva(
  "flex min-h-9 w-full rounded-lg text-[13px] leading-5 transition-colors duration-300 disabled:cursor-not-allowed disabled:opacity-50 placeholder:text-gray-7 text-gray-12",
  {
    variants: {
      variant: {
        default: [
          "border border-gray-5 hover:border-gray-8 bg-gray-2",
          "focus:border focus:border-accent-12 focus:ring-4 focus:ring-gray-5 focus-visible:outline-none focus:ring-offset-0",
        ],
        success: [
          "border border-success-9 hover:border-success-10 bg-gray-2",
          "focus:border-success-8 focus:ring-2 focus:ring-success-2 focus-visible:outline-none",
        ],
        warning: [
          "border border-warning-9 hover:border-warning-10 bg-gray-2",
          "focus:border-warning-8 focus:ring-2 focus:ring-warning-2 focus-visible:outline-none",
        ],
        error: [
          "border border-error-9 hover:border-error-10 bg-gray-2",
          "focus:border-error-8 focus:ring-2 focus:ring-error-2 focus-visible:outline-none",
        ],
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

const textareaWrapperVariants = cva("relative flex items-center w-full", {
  variants: {
    variant: {
      default: "text-gray-11",
      success: "text-success-11",
      warning: "text-warning-11",
      error: "text-error-11",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

// Hack to populate fumadocs' AutoTypeTable
export type DocumentedTextareaProps = VariantProps<typeof textareaVariants> & {
  leftIcon?: React.ReactNode;
  rightIcon?: React.ReactNode;
  wrapperClassName?: string;
};

export type TextareaProps = DocumentedTextareaProps &
  React.TextareaHTMLAttributes<HTMLTextAreaElement>;

export const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, variant, leftIcon, rightIcon, wrapperClassName, ...props }, ref) => {
    return (
      <div className={cn(textareaWrapperVariants({ variant }), wrapperClassName)}>
        {leftIcon && (
          <div className="absolute left-3 top-3 flex items-start pointer-events-none">
            {leftIcon}
          </div>
        )}
        <textarea
          className={cn(
            textareaVariants({ variant, className }),
            "px-3 py-2",
            leftIcon && "pl-9",
            rightIcon && "pr-9",
          )}
          ref={ref}
          {...props}
        />
        {rightIcon && <div className="absolute right-3 top-3 flex items-start">{rightIcon}</div>}
      </div>
    );
  },
);
Textarea.displayName = "Textarea";
