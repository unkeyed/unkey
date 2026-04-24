"use client";

import * as PopoverPrimitive from "@radix-ui/react-popover";
import type * as React from "react";
import { cn } from "~/lib/utils";

export const Popover = PopoverPrimitive.Root;
export const PopoverTrigger = PopoverPrimitive.Trigger;
export const PopoverAnchor = PopoverPrimitive.Anchor;

export function PopoverContent({
  className,
  align = "start",
  sideOffset = 6,
  ...props
}: React.ComponentPropsWithoutRef<typeof PopoverPrimitive.Content>) {
  return (
    <PopoverPrimitive.Portal>
      <PopoverPrimitive.Content
        align={align}
        sideOffset={sideOffset}
        className={cn(
          "z-50 min-w-56 rounded-md border border-gray-6 bg-background p-3 text-gray-12 shadow-md outline-hidden",
          "ease-[cubic-bezier(0.4,0,0.2,1)]",
          "data-[state=open]:duration-200 data-[state=open]:animate-in",
          "data-[state=open]:fade-in-0 data-[state=open]:zoom-in-95",
          className,
        )}
        {...props}
      />
    </PopoverPrimitive.Portal>
  );
}
