import { cn } from "@/lib/utils";
import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { forwardRef } from "react";

const badgeVariants = cva("inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs", {
  variants: {
    variant: {
      primary: "border-transparent bg-grayA-3 text-grayA-11 hover:bg-grayA-4",
      secondary: "border-border bg-grayA-2 text-grayA-11 hover:bg-grayA-3 font-normal",
      success: "border-transparent bg-grayA-3 text-grayA-11 hover:bg-grayA-4",
      warning: "border-transparent bg-warningA-3 text-warningA-11 hover:bg-warningA-4",
      blocked: "border-transparent bg-orangeA-3 text-orangeA-11 hover:bg-orangeA-4",
      error: "border-transparent bg-errorA-3 text-errorA-11 hover:bg-errorA-4",
    },
    size: {
      DEFAULT: "",
      sm: "px-1.5 py-0.5",
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

const Badge = forwardRef<HTMLDivElement, BadgeProps>(
  ({ className, variant, size, font, ...props }, ref) => {
    return (
      <div ref={ref} className={cn(badgeVariants({ variant, size, font }), className)} {...props} />
    );
  },
);

Badge.displayName = "Badge";

export { Badge, badgeVariants };
