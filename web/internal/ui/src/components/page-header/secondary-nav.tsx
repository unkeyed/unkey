import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import type * as React from "react";
import { cn } from "../../lib/utils";

function SecondaryNav({ className, ...props }: React.ComponentProps<"nav">) {
  return (
    <nav
      className={cn(
        "flex shrink-0 gap-1 overflow-x-auto border-b border-grayA-4 px-3 py-2",
        "md:w-60 md:flex-col md:gap-4 md:overflow-x-visible md:overflow-y-auto md:border-r md:border-b-0 md:px-3 md:py-5",
        className,
      )}
      {...props}
    />
  );
}

function SecondaryNavTitle({ className, ...props }: React.ComponentProps<"h2">) {
  return (
    <h2
      className={cn(
        "hidden md:block px-2 text-[15px] font-semibold tracking-tight leading-tight text-accent-12 m-0",
        className,
      )}
      {...props}
    />
  );
}

function SecondaryNavGroup({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("contents md:flex md:flex-col md:gap-1", className)} {...props} />;
}

const secondaryNavItemVariants = cva(
  "flex items-center shrink-0 whitespace-nowrap rounded-md px-2 py-1.5 text-[13px] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-grayA-7",
  {
    variants: {
      active: {
        true: "bg-grayA-3 text-accent-12 font-medium",
        false: "text-accent-11 hover:bg-grayA-3 hover:text-accent-12",
      },
    },
    defaultVariants: {
      active: false,
    },
  },
);

type SecondaryNavItemProps = React.ComponentProps<"a"> &
  VariantProps<typeof secondaryNavItemVariants> & {
    asChild?: boolean;
  };

function SecondaryNavItem({ className, active, asChild = false, ...props }: SecondaryNavItemProps) {
  const Comp = asChild ? Slot : "a";
  return (
    <Comp
      aria-current={active ? "page" : undefined}
      className={cn(secondaryNavItemVariants({ active }), className)}
      {...props}
    />
  );
}

export {
  SecondaryNav,
  SecondaryNavTitle,
  SecondaryNavGroup,
  SecondaryNavItem,
  secondaryNavItemVariants,
};
