import type React from "react";
import type { PropsWithChildren } from "react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "./tooltip";

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
  style,
  triggerClassName,
}: PropsWithChildren<{
  delayDuration?: number;
  content: React.ReactNode;
  position?: TooltipPosition;
  disabled?: boolean;
  asChild?: boolean;
  className?: string;
  style?: React.CSSProperties;
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
          className={className}
          style={style}
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
