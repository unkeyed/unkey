"use client";

import { PreviewCard as HoverCardPrimitive } from "@base-ui/react/preview-card";
import type * as React from "react";
import { cn } from "../lib/utils";

const HoverCard = HoverCardPrimitive.Root;
const HoverCardTrigger = HoverCardPrimitive.Trigger;

function HoverCardContent({
  className,
  align = "center",
  alignOffset,
  side,
  sideOffset = 4,
  ref,
  ...props
}: HoverCardPrimitive.Popup.Props &
  Pick<HoverCardPrimitive.Positioner.Props, "align" | "alignOffset" | "side" | "sideOffset"> & {
    ref?: React.Ref<React.ComponentRef<typeof HoverCardPrimitive.Popup>>;
  }) {
  return (
    <HoverCardPrimitive.Portal>
      <HoverCardPrimitive.Positioner
        className="isolate z-200"
        align={align}
        alignOffset={alignOffset}
        side={side}
        sideOffset={sideOffset}
      >
        <HoverCardPrimitive.Popup
          ref={ref}
          className={cn(
            "z-200 w-64 rounded-lg border border-grayA-4 bg-gray-2 p-4 text-gray-12 shadow-md outline-none",
            "transition-[opacity,scale,translate] data-starting-style:opacity-0 data-starting-style:scale-95 data-ending-style:opacity-0 data-ending-style:scale-95",
            "data-[side=bottom]:data-starting-style:-translate-y-2 data-[side=left]:data-starting-style:translate-x-2 data-[side=right]:data-starting-style:-translate-x-2 data-[side=top]:data-starting-style:translate-y-2",
            className,
          )}
          {...props}
        />
      </HoverCardPrimitive.Positioner>
    </HoverCardPrimitive.Portal>
  );
}

export { HoverCard, HoverCardTrigger, HoverCardContent };
