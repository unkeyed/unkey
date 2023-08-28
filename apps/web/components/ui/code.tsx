import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const codeVariants = cva(
  "inlineflex font-mono items-center rounded-md border border-input bg-transparent px-2.5 py-2 text-sm font-mono  transition-colors focus:outline-none focus:ring-2 focus:ring-border focus:ring-offset-2",
  {
    variants: {
      variant: {
        default: "bg-muted text-primary hover:border-primary",

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

export { Code, codeVariants };
