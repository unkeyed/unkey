import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "~/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 font-medium text-xs transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12 [&_svg]:pointer-events-none [&_svg]:size-3 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default: "border-transparent bg-gray-3 text-gray-12 [a&]:hover:bg-gray-4",
        secondary: "border-gray-6 bg-background text-gray-11 [a&]:hover:bg-gray-2",
        success: "border-transparent bg-successA-3 text-successA-11 [a&]:hover:bg-successA-4",
        warning: "border-transparent bg-warningA-3 text-warningA-11 [a&]:hover:bg-warningA-3",
        error: "border-transparent bg-errorA-3 text-errorA-11 [a&]:hover:bg-errorA-4",
        outline: "border-gray-6 text-gray-12 [a&]:hover:bg-gray-2",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

type BadgeProps = React.HTMLAttributes<HTMLSpanElement> &
  VariantProps<typeof badgeVariants> & {
    asChild?: boolean;
  };

export function Badge({ className, variant, asChild, ...props }: BadgeProps) {
  const Comp = asChild ? Slot : "span";
  return <Comp className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export { badgeVariants };
export type { BadgeProps };
