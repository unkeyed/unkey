"use client";

import * as SwitchPrimitives from "@radix-ui/react-switch";
import * as React from "react";

import { cn } from "@/lib/utils";

const Switch = React.forwardRef<
  React.ElementRef<typeof SwitchPrimitives.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitives.Root> & {
    size?: "default" | "sm";
  }
>(({ className, size = "default", ...props }, ref) => (
  <SwitchPrimitives.Root
    data-size={size}
    className={cn(
      "peer group/switch inline-flex shrink-0 cursor-pointer items-center rounded-full p-0.5 transition-[background-color,box-shadow] duration-150 ease-out focus-visible:outline-2 focus-visible:outline-accent-8 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-50 disabled:grayscale data-[state=checked]:bg-primary data-[state=unchecked]:bg-grayA-5 data-[size=default]:h-6 data-[size=default]:w-10 data-[size=sm]:h-5 data-[size=sm]:w-10",
      className,
    )}
    {...props}
    ref={ref}
  >
    <SwitchPrimitives.Thumb
      className={cn(
        "pointer-events-none block rounded-full bg-background shadow-sm transition-transform duration-150 ease-out data-[state=unchecked]:translate-x-0 dark:data-[state=checked]:bg-primary-foreground dark:data-[state=unchecked]:bg-foreground",
        "group-data-[size=default]/switch:h-5 group-data-[size=default]/switch:w-5 group-data-[size=default]/switch:data-[state=checked]:translate-x-4",
        "group-data-[size=sm]/switch:h-4 group-data-[size=sm]/switch:w-4 group-data-[size=sm]/switch:data-[state=checked]:translate-x-5",
      )}
    />
  </SwitchPrimitives.Root>
));
Switch.displayName = SwitchPrimitives.Root.displayName;

export { Switch };
