import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";

import { cn } from "@/lib/utils";

const badgeVariants = cva("inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs", {
  variants: {
    variant: {
      primary: "border-transparent bg-primary text-primary-foreground hover:bg-primary/80",
      secondary:
        "border-border bg-secondary text-secondary-foreground hover:bg-secondary/80 font-normal",
      alert: "border-transparent bg-alert text-alert-foreground hover:bg-alert/80",
    },
    size: {
      DEFAULT: "",
      sm: "px-1 py-0.4",
    },
    font: {
      mono: "font-mono",
    },
  },

  defaultVariants: {
    variant: "primary",
  },
});

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, size, font, ...props }: BadgeProps) {
  return <div className={cn(badgeVariants({ variant, size, font }), className)} {...props} />;
}

export { Badge, badgeVariants };

// font-mono text-sm text-secondary inline-flex rounded-sm bg-gray-50 px-1.5 shadow-[0_0_0_1px,0_1px_0] shadow-gray-200/60
