import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import type { PropsWithChildren } from "react";

type TooltipPosition = {
  side?: "top" | "right" | "bottom" | "left";
  align?: "start" | "center" | "end";
  sideOffset?: number;
};

export const RatelimitOverviewTooltip = ({
  content,
  children,
  position,
  disabled = false,
  asChild = false,
}: PropsWithChildren<{
  content: React.ReactNode;
  position?: TooltipPosition;
  disabled?: boolean;
  asChild?: boolean;
}>) => {
  return (
    <TooltipProvider>
      <Tooltip open={disabled ? false : undefined}>
        <TooltipTrigger asChild={asChild}>{children}</TooltipTrigger>
        <TooltipContent
          className="bg-gray-12 text-gray-1 px-3 py-2 border border-accent-6 shadow-md font-medium text-xs"
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
