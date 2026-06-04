import { Slot } from "@radix-ui/react-slot";
import { type VariantProps, cva } from "class-variance-authority";
import * as React from "react";
import { cn } from "../../lib/utils";

/**
 * A section-level navigation rail. On desktop it renders as a left sidebar; on
 * smaller screens it collapses to a horizontal scrolling bar. Items are links,
 * so inject your router's link via `asChild`:
 *
 * ```tsx
 * <SecondaryNav aria-label="Settings">
 *   <SecondaryNavTitle>Settings</SecondaryNavTitle>
 *   <SecondaryNavGroup>
 *     <SecondaryNavItem asChild active>
 *       <Link href="/settings/general">General</Link>
 *     </SecondaryNavItem>
 *   </SecondaryNavGroup>
 * </SecondaryNav>
 * ```
 */
const SecondaryNav = React.forwardRef<HTMLElement, React.HTMLAttributes<HTMLElement>>(
  ({ className, ...props }, ref) => (
    <nav
      ref={ref}
      className={cn(
        "flex shrink-0 gap-1 overflow-x-auto border-b border-grayA-4 px-3 py-2",
        "md:w-60 md:flex-col md:gap-4 md:overflow-x-visible md:overflow-y-auto md:border-r md:border-b-0 md:px-3 md:py-5",
        className,
      )}
      {...props}
    />
  ),
);
SecondaryNav.displayName = "SecondaryNav";

/** The section heading shown above the items. Hidden on the mobile bar. */
const SecondaryNavTitle = React.forwardRef<
  HTMLHeadingElement,
  React.HTMLAttributes<HTMLHeadingElement>
>(({ className, ...props }, ref) => (
  <h2
    ref={ref}
    className={cn(
      "hidden md:block px-2 text-[15px] font-semibold tracking-tight leading-tight text-accent-12 m-0",
      className,
    )}
    {...props}
  />
));
SecondaryNavTitle.displayName = "SecondaryNavTitle";

/**
 * Groups related items. On the mobile bar the group dissolves so its items
 * flow inline.
 */
const SecondaryNavGroup = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("contents md:flex md:flex-col md:gap-1", className)} {...props} />
  ),
);
SecondaryNavGroup.displayName = "SecondaryNavGroup";

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

type SecondaryNavItemProps = React.AnchorHTMLAttributes<HTMLAnchorElement> &
  VariantProps<typeof secondaryNavItemVariants> & {
    /** Render the passed child as the item (e.g. a framework `Link`). */
    asChild?: boolean;
  };

const SecondaryNavItem = React.forwardRef<HTMLAnchorElement, SecondaryNavItemProps>(
  ({ className, active, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "a";
    return (
      <Comp
        ref={ref}
        aria-current={active ? "page" : undefined}
        className={cn(secondaryNavItemVariants({ active }), className)}
        {...props}
      />
    );
  },
);
SecondaryNavItem.displayName = "SecondaryNavItem";

export {
  SecondaryNav,
  SecondaryNavTitle,
  SecondaryNavGroup,
  SecondaryNavItem,
  secondaryNavItemVariants,
};
