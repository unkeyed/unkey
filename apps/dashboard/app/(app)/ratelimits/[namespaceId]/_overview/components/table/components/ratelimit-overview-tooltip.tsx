import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import type { PropsWithChildren } from "react";

export const RatelimitOverviewTooltip = ({
  content,
  children,
}: PropsWithChildren<{
  content: React.ReactNode;
}>) => {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>{children}</TooltipTrigger>
        <TooltipContent
          className="bg-gray-12 text-gray-1 px-3 py-2 border border-accent-6 shadow-md font-medium text-xs"
          side="right"
        >
          {content}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
