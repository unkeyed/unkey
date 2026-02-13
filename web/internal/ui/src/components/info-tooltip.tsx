// biome-ignore lint: React in this context is used throughout, so biome will change to types because no APIs are used even though React is needed.
import React, { type PropsWithChildren } from "react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./tooltip";
import { cn } from "../lib/utils";

const baseVariant =
  "px-3 py-2 text-xs font-medium shadow-md rounded-lg focus:border focus:border-accent-12 focus:ring-2 focus:ring-grayA-4 focus-visible:outline-hidden focus:ring-offset-0";
const variants = {
  primary: ["border border-grayA-4 bg-white dark:bg-black"],
  inverted: ["bg-black dark:bg-white text-gray-1 border border-grayA-4"],
  secondary: ["border dark:border-gray-12 text-gray-12 text-sm"],
  muted: ["border border-grayA-4 text-gray-12 text-sm"],
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
          className={cn(baseVariant, variants[variant], className)}
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

InfoTooltip.displayName = "InfoTooltip";
export { InfoTooltip };
