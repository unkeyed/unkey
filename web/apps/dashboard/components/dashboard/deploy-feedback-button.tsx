"use client";

import { useFeedback } from "@/components/dashboard/feedback-component";
import { Chats } from "@unkey/icons";
import { Button } from "@unkey/ui";

export function DeployFeedbackButton() {
  const { openFeedback } = useFeedback();

  return (
    <Button variant="outline" size="md" onClick={() => openFeedback(true, "feedback")}>
      <Chats className="size-4" />
      Feedback
    </Button>
  );
}
