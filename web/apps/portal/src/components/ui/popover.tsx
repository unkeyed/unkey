"use client";

import { Popover as PopoverPrimitive } from "@base-ui/react/popover";
import { cn } from "~/lib/utils";

export const Popover = PopoverPrimitive.Root;
export const PopoverTrigger = PopoverPrimitive.Trigger;

type PopoverContentProps = PopoverPrimitive.Popup.Props &
  Pick<PopoverPrimitive.Positioner.Props, "align" | "alignOffset" | "side" | "sideOffset">;

export function PopoverContent({
  className,
  align = "start",
  alignOffset,
  side,
  sideOffset = 6,
  ...props
}: PopoverContentProps) {
  return (
    <PopoverPrimitive.Portal>
      <PopoverPrimitive.Positioner
        className="isolate z-50"
        align={align}
        alignOffset={alignOffset}
        side={side}
        sideOffset={sideOffset}
      >
        <PopoverPrimitive.Popup
          className={cn(
            "z-50 min-w-56 rounded-md border border-gray-6 bg-background p-3 text-gray-12 shadow-md outline-hidden",
            "origin-(--transform-origin) transition-[opacity,scale] duration-200 ease-[cubic-bezier(0.4,0,0.2,1)]",
            "data-starting-style:scale-95 data-ending-style:opacity-0 data-starting-style:opacity-0",
            className,
          )}
          {...props}
        />
      </PopoverPrimitive.Positioner>
    </PopoverPrimitive.Portal>
  );
}
