"use client";

import { Popover as PopoverPrimitive } from "@base-ui/react/popover";
import * as React from "react";
import { cn } from "../../lib/utils";

const Popover = PopoverPrimitive.Root;
// Bare re-export: Base UI's default `nativeButton={true}` is correct for the
// common case (trigger renders a real <button>/<Button>). Consumers that render
// a non-button element pass `nativeButton={false}` at the call site.
const PopoverTrigger = PopoverPrimitive.Trigger;
// Note: when the root has `modal={true}`, Base UI only enables its focus trap
// if a <Popover.Close> is rendered inside the popup (an sr-only one suffices).
const PopoverClose = PopoverPrimitive.Close;

type PopoverContentProps = PopoverPrimitive.Popup.Props &
  Pick<
    PopoverPrimitive.Positioner.Props,
    "align" | "alignOffset" | "side" | "sideOffset" | "anchor"
  >;

const PopoverContent = React.forwardRef<
  React.ComponentRef<typeof PopoverPrimitive.Popup>,
  PopoverContentProps
>(({ className, align = "center", alignOffset, side, sideOffset = 4, anchor, ...props }, ref) => (
  <PopoverPrimitive.Portal>
    <PopoverPrimitive.Positioner
      className="isolate z-200"
      align={align}
      alignOffset={alignOffset}
      side={side}
      sideOffset={sideOffset}
      anchor={anchor}
    >
      <PopoverPrimitive.Popup
        ref={ref}
        className={cn(
          "z-200 w-72 rounded-lg border border-grayA-4 bg-gray-2 p-4 text-gray-12 shadow-md outline-none",
          "transition-[opacity,scale,translate] data-starting-style:opacity-0 data-starting-style:scale-95 data-ending-style:opacity-0 data-ending-style:scale-95",
          "data-[side=bottom]:data-starting-style:-translate-y-2 data-[side=left]:data-starting-style:translate-x-2 data-[side=right]:data-starting-style:-translate-x-2 data-[side=top]:data-starting-style:translate-y-2",
          className,
        )}
        {...props}
      />
    </PopoverPrimitive.Positioner>
  </PopoverPrimitive.Portal>
));
PopoverContent.displayName = "PopoverContent";

export { Popover, PopoverTrigger, PopoverClose, PopoverContent };
export type { PopoverContentProps };
