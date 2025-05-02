// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React, { type PropsWithChildren } from "react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./tooltip";
import { cn } from "../lib/utils";

const variants = {
  primary: [
    "border border-grayA-4 hover:border-gray-8 bg-gray-2 dark:bg-black",
    "focus:border focus:border-accent-12 focus:ring-2 focus:ring-grayA-4 focus-visible:outline-none focus:ring-offset-0",
  ],
  secondary: [
    "bg-gray-1 text-accent-12 border border-grayA-4 px-3 py-2 text-xs font-medium shadow-md rounded-lg",
  ],
} as const;

type TooltipVariant = keyof typeof variants;

type TooltipPosition = {
  side?: "top" | "right" | "bottom" | "left";
  align?: "start" | "center" | "end";
  sideOffset?: number;
};

const InfoTooltip = ({
  delayDuration,
  content,
  children,
  position,
  disabled = false,
  asChild = false,
  className,
  variant = "primary",
  triggerClassName,
}: PropsWithChildren<{
  variant?: TooltipVariant;
  delayDuration?: number;
  content: React.ReactNode;
  position?: TooltipPosition;
  disabled?: boolean;
  asChild?: boolean;
  className?: string;
  triggerClassName?: string;
}>) => {
  return (
    <TooltipProvider delayDuration={delayDuration ?? undefined}>
      <Tooltip open={disabled ? false : undefined}>
        <TooltipTrigger asChild={asChild} className={triggerClassName}>
          {children}
        </TooltipTrigger>
        <TooltipContent
          className={cn(variants[variant], className)}
          side={position?.side || "right"}
          align={position?.align || "center"}
          sideOffset={position?.sideOffset}
        >
          {content}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};

export { InfoTooltip };
