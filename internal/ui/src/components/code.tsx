"use client";
import { type VariantProps, cva } from "class-variance-authority";
import React from "react";
import { cn } from "../lib/utils";

const codeVariants = cva(
  "flex w-full px-4 border rounded-xl ring-0 flex items-start justify-between ph-no-capture whitespace-pre-wrap break-all ",
  {
    variants: {
      variant: {
        default:
          "border-grayA-5 focus:outline-none focus:ring-0 bg-white dark:bg-black text-[11px] py-2",
        ghost: "border-none bg-transparent text-[11px] py-2",
        legacy:
          "text-primary bg-background-subtle rounded-md hover:border-primary focus:outline-none focus:ring-0 border-grayA-4",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

export interface CodeProps
  extends React.HTMLAttributes<HTMLPreElement>,
    VariantProps<typeof codeVariants> {
  className?: string;
  buttonsClassName?: string;
  preClassName?: string;
  variant?: "default" | "ghost" | "legacy";
  copyButton?: React.ReactNode;
  visibleButton?: React.ReactNode;
}

function Code({
  className,
  variant,
  copyButton,
  visibleButton,
  buttonsClassName,
  preClassName,
  ...props
}: CodeProps) {
  return (
    <div className={cn(codeVariants({ variant }), className)}>
      <pre
        className={cn(
          "border-none bg-transparent focus:outline-none focus:ring-0 pr-2 pt-2",
          preClassName
        )}
        {...props}
      />
      <div
        className={cn(
          "flex items-center justify-between gap-2 flex-shrink-0 mt-1",
          buttonsClassName
        )}
      >
        {visibleButton}
        {copyButton}
      </div>
    </div>
  );
}

Code.displayName = "Code";

export { Code, codeVariants };
