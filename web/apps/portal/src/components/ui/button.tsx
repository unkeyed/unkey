import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "~/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md font-medium text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-gray-12 focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "bg-[var(--portal-btn,var(--color-gray-12))] text-white transition-opacity hover:opacity-90",
        outline: "border border-primary/15 bg-background shadow-xs hover:bg-gray-2",
        ghost: "text-gray-12 hover:bg-gray-3",
        destructive: "bg-error-9 text-white hover:bg-error-11",
      },
      size: {
        default: "h-8 px-3 has-[>svg:last-child]:pr-2 has-[>svg:first-child]:pl-2",
        sm: "h-6 px-3 text-xs has-[>svg:last-child]:pr-2 has-[>svg:first-child]:pl-2",
        icon: "h-8 w-8 p-0",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
  };

export function Button({ className, variant, size, asChild, ...props }: ButtonProps) {
  const Comp = asChild ? Slot : "button";
  return <Comp className={cn(buttonVariants({ variant, size }), className)} {...props} />;
}

export { buttonVariants };
export type { ButtonProps };
