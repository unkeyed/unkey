import type React from "react";
import type { PropsWithChildren } from "react";
import { cn } from "../lib/utils";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./tooltip";

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
    <TooltipProvider delay={delayDuration ?? undefined}>
      {/* Base UI's own `disabled` prop keeps the root uncontrolled and clears
          open state properly — no controlled/uncontrolled flip, no stale open
          state surviving a disabled toggle. */}
      <Tooltip disabled={disabled}>
        {asChild ? (
          <TooltipTrigger className={triggerClassName} render={children as React.ReactElement} />
        ) : (
          <TooltipTrigger className={triggerClassName}>{children}</TooltipTrigger>
        )}
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
