import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";

import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-md font-medium text-sm max-sm:text-xs transition-colors focus-visible:outline-none  disabled:pointer-events-none	 duration-200",
  {
    variants: {
      variant: {
        primary:
          "bg-primary text-primary-foreground hover:bg-secondary hover:text-secondary-foreground border border-primary",
        secondary:
          " text-secondary-foreground hover:bg-primary hover:text-primary-foreground border border-border",
        outline: "border border-input bg-background hover:bg-accent hover:text-accent-foreground",
        alert:
          "bg-background border border-[#b80f07] text-[#b80f07] hover:bg-[#b80f07] hover:text-white",
        disabled: "text-secondary-foreground bg-secondary border border-border opacity-50",

        ghost: "hover:bg-gray-200 hover:text-gray-900",
        link: "text-subtle underline-offset-4 hover:underline",
      },
      size: {
        default: "h-8 px-3 py-1.5",
        sm: "h-6 px-3 py-2",
        lg: "h-10 px-8",
        xl: "h-12 px-8",
        icon: "h-8 w-8",
        block: "h-8 px-3 py-1.5  w-full",
      },
    },

    defaultVariants: {
      variant: "primary",
      size: "default",
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />
    );
  },
);
Button.displayName = "Button";

export { Button, buttonVariants };
