"use client";

import { Menu as DropdownMenuPrimitive } from "@base-ui/react/menu";
import { type VariantProps, cva } from "class-variance-authority";
import { cn } from "~/lib/utils";

export const DropdownMenu = DropdownMenuPrimitive.Root;
export const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger;
export const DropdownMenuGroup = DropdownMenuPrimitive.Group;
export const DropdownMenuPortal = DropdownMenuPrimitive.Portal;

type DropdownMenuContentProps = DropdownMenuPrimitive.Popup.Props &
  Pick<DropdownMenuPrimitive.Positioner.Props, "align" | "alignOffset" | "side" | "sideOffset">;

export function DropdownMenuContent({
  className,
  align,
  alignOffset,
  side,
  sideOffset = 4,
  ...props
}: DropdownMenuContentProps) {
  return (
    <DropdownMenuPrimitive.Portal>
      <DropdownMenuPrimitive.Positioner
        className="isolate z-50"
        align={align}
        alignOffset={alignOffset}
        side={side}
        sideOffset={sideOffset}
      >
        <DropdownMenuPrimitive.Popup
          className={cn(
            "z-50 min-w-40 overflow-hidden rounded-lg border border-primary/15 bg-background p-1 shadow-md outline-none",
            "origin-(--transform-origin) transition-[opacity,scale] data-ending-style:scale-95 data-starting-style:scale-95 data-ending-style:opacity-0 data-starting-style:opacity-0",
            className,
          )}
          {...props}
        />
      </DropdownMenuPrimitive.Positioner>
    </DropdownMenuPrimitive.Portal>
  );
}

const dropdownMenuItemVariants = cva(
  [
    "relative flex cursor-pointer select-none items-center gap-2 rounded-md px-2 py-1.5 text-sm outline-hidden transition-colors",
    "data-disabled:pointer-events-none data-disabled:opacity-50",
    "[&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0",
  ],
  {
    variants: {
      variant: {
        default: "text-gray-12 data-highlighted:bg-gray-3",
        destructive: "text-error-11 data-highlighted:bg-error-3",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

type DropdownMenuItemProps = DropdownMenuPrimitive.Item.Props &
  VariantProps<typeof dropdownMenuItemVariants>;

export function DropdownMenuItem({ className, variant, ...props }: DropdownMenuItemProps) {
  return (
    <DropdownMenuPrimitive.Item
      className={cn(dropdownMenuItemVariants({ variant }), className)}
      {...props}
    />
  );
}

// Menu.GroupLabel wires the group's aria-labelledby; every DropdownMenuLabel
// call site must therefore sit inside a DropdownMenuGroup.
export function DropdownMenuLabel({ className, ...props }: DropdownMenuPrimitive.GroupLabel.Props) {
  return (
    <DropdownMenuPrimitive.GroupLabel
      className={cn("px-2 py-1.5 font-medium text-gray-11 text-xs", className)}
      {...props}
    />
  );
}

export function DropdownMenuSeparator({
  className,
  ...props
}: DropdownMenuPrimitive.Separator.Props) {
  return (
    <DropdownMenuPrimitive.Separator
      className={cn("-mx-2 my-1 h-px bg-gray-6", className)}
      {...props}
    />
  );
}

export { dropdownMenuItemVariants };
