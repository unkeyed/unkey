"use client";

import { useFeedback } from "@/components/dashboard/feedback-component";
import { IconChatsOutline18 } from "@unkey/icons";
import { Tooltip, TooltipContent, TooltipTrigger } from "@unkey/ui";
import { cn } from "cn";
import { TOP_NAV_ICON_BUTTON_CLASS, TOP_NAV_TOOLTIP_CLASS } from "./icon-button";

export function TopNavFeedbackButton({ className }: { className?: string }) {
  const { openFeedback } = useFeedback();
  return (
    <Tooltip>
      <TooltipTrigger
        aria-label="Feedback"
        onClick={() => openFeedback(true, "feedback")}
        className={cn(TOP_NAV_ICON_BUTTON_CLASS, className)}
      >
        <IconChatsOutline18 className="size-3.5" />
      </TooltipTrigger>
      <TooltipContent side="bottom" align="center" className={TOP_NAV_TOOLTIP_CLASS}>
        Feedback
      </TooltipContent>
    </Tooltip>
  );
}
