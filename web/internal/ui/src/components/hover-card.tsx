"use client";

import { PreviewCard as HoverCardPrimitive } from "@base-ui/react/preview-card";
import type * as React from "react";
import { popupTransition } from "../lib/popup";
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
            "z-200 w-64 rounded-lg bg-raised p-4 text-gray-12 shadow-floating outline-none",
            popupTransition,
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
