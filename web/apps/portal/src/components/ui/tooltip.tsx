"use client";

import { Tooltip as TooltipPrimitive } from "@base-ui/react/tooltip";
import { cn } from "~/lib/utils";

export const TooltipProvider = TooltipPrimitive.Provider;
export const Tooltip = TooltipPrimitive.Root;
export const TooltipTrigger = TooltipPrimitive.Trigger;

type TooltipContentProps = TooltipPrimitive.Popup.Props &
  Pick<TooltipPrimitive.Positioner.Props, "side" | "align" | "alignOffset" | "sideOffset">;

export function TooltipContent({
  className,
  side,
  align,
  alignOffset,
  sideOffset = 6,
  ...props
}: TooltipContentProps) {
  return (
    <TooltipPrimitive.Portal>
      <TooltipPrimitive.Positioner
        className="isolate z-50"
        side={side}
        align={align}
        alignOffset={alignOffset}
        sideOffset={sideOffset}
      >
        <TooltipPrimitive.Popup
          className={cn(
            "z-50 overflow-hidden rounded-md bg-gray-12 px-2 py-1 text-background text-xs shadow-md",
            "origin-(--transform-origin) transition-[opacity,scale] duration-150 data-ending-style:scale-95 data-starting-style:scale-95 data-ending-style:opacity-0 data-starting-style:opacity-0",
            className,
          )}
          {...props}
        />
      </TooltipPrimitive.Positioner>
    </TooltipPrimitive.Portal>
  );
}
