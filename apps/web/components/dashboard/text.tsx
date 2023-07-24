import { VariantProps, cva } from "class-variance-authority";
import * as React from "react";

import { cn } from "@/lib/utils";

const textVariants = cva("", {
  variants: {
    variant: {
      default: "leading-7 text-stone-900",
      code: "relative rounded bg-stone-100 py-[0.2rem] px-[0.3rem] font-mono text-sm font-semibold text-stone-900",
      lead: "text-stone-700",

      subtle: "text-stone-500",
    },
    size: {
      xs: "text-xs leading-none",
      sm: "text-sm leading-none",
      lg: "text-lg font-semibold",
      xl: "text-xl font-semibold",
    },
  },
  defaultVariants: {
    variant: "default",
  },
});

export interface TextProps
  extends React.HTMLAttributes<HTMLElement>,
    VariantProps<typeof textVariants> {}

const Text = React.forwardRef<HTMLElement, TextProps>(
  ({ variant, size, children, ...props }, ref) => {
    switch (variant) {
      case "code":
        return (
          <code className={cn(textVariants({ variant, size }))} ref={ref} {...props}>
            {children}
          </code>
        );

      default:
        return (
          <p className={cn(textVariants({ variant, size }))} {...props}>
            {children}
          </p>
        );
    }
  },
);
Text.displayName = "Text";

export { Text, textVariants };
