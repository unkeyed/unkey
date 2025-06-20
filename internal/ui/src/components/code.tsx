"use client";
import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "../lib/utils";

const codeVariants = cva(
  "inline-flex font-mono items-center rounded-md border border-border bg-transparent px-2.5 py-2 text-sm transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default: " text-primary bg-background-subtle hover:border-primary",

        outline: "text-foreground",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

export interface CodeProps
  extends React.HTMLAttributes<HTMLPreElement>,
    VariantProps<typeof codeVariants> {}

function Code({ className, variant, ...props }: CodeProps) {
  return <pre className={cn(codeVariants({ variant }), className)} {...props} />;
}

Code.displayName = "Code";

export { Code, codeVariants };
