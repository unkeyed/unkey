"use client";

import { PreviewCard as HoverCardPrimitive } from "@base-ui/react/preview-card";
import * as React from "react";
import { cn } from "../lib/utils";

const HoverCard = HoverCardPrimitive.Root;
const HoverCardTrigger = HoverCardPrimitive.Trigger;

const HoverCardContent = React.forwardRef<
  React.ComponentRef<typeof HoverCardPrimitive.Popup>,
  HoverCardPrimitive.Popup.Props &
    Pick<HoverCardPrimitive.Positioner.Props, "align" | "alignOffset" | "side" | "sideOffset">
>(({ className, align = "center", alignOffset, side, sideOffset = 4, ...props }, ref) => (
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
));
HoverCardContent.displayName = "HoverCardContent";

export { HoverCard, HoverCardTrigger, HoverCardContent };
