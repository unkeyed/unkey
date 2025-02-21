import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@unkey/ui";
import type { PropsWithChildren } from "react";

export const InputTooltip = ({ desc, children }: PropsWithChildren<{ desc: string }>) => {
  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>{children}</TooltipTrigger>
        <TooltipContent className="bg-gray-12 text-gray-1 px-3 py-2 border border-accent-6 shadow-md font-medium text-xs">
          <p className="text-sm">{desc}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
