"use client";

import { Button } from "@unkey/ui";
import { IconChatsOutline18 } from "nucleo-ui-outline-18";
import { useFeedback } from "@/components/dashboard/feedback-component";

export function TopNavFeedbackButton({ className }: { className?: string }) {
  const { openFeedback } = useFeedback();
  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => openFeedback(true, "feedback")}
      className={className}
    >
      <IconChatsOutline18 className="size-4" />
      Feedback
    </Button>
  );
}
