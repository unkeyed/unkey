"use client";

import { useFeedback } from "@/components/dashboard/feedback-component";
import { Chats } from "@unkey/icons";
import { Button } from "@unkey/ui";

export function TopNavFeedbackButton({ className }: { className?: string }) {
  const { openFeedback } = useFeedback();
  return (
    <Button
      variant="outline"
      size="sm"
      onClick={() => openFeedback(true, "feedback")}
      className={className}
    >
      <Chats className="size-4" />
      Feedback
    </Button>
  );
}
