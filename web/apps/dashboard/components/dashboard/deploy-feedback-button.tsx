"use client";

import { useFeedback } from "@/components/dashboard/feedback-component";
import { Chats } from "@unkey/icons";
import { Button } from "@unkey/ui";

export function DeployFeedbackButton() {
  const { openFeedback } = useFeedback();

  return (
    <Button variant="outline" size="sm" onClick={() => openFeedback(true, "feedback")}>
      <Chats iconSize="sm-medium" />
      Feedback
    </Button>
  );
}
