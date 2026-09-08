"use client";

import { Separator as SeparatorPrimitive } from "@base-ui/react/separator";
import type * as React from "react";

import { cn } from "../lib/utils";

function Separator({
  className,
  orientation = "horizontal",
  ref,
  ...props
}: React.ComponentPropsWithoutRef<typeof SeparatorPrimitive> & {
  ref?: React.Ref<React.ComponentRef<typeof SeparatorPrimitive>>;
}) {
  return (
    <SeparatorPrimitive
      ref={ref}
      orientation={orientation}
      className={cn(
        "shrink-0 bg-gray-4",
        orientation === "horizontal" ? "h-px w-full" : "w-px",
        className,
      )}
      {...props}
    />
  );
}

export { Separator };
