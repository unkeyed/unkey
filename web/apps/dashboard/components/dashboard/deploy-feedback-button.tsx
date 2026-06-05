"use client";

import { useFlag } from "@/lib/flags/provider";
import { Chats } from "@unkey/icons";
import { Button } from "@unkey/ui";

const USERJOT_FEEDBACK_URL = "https://feedback.unkey.com/";

export function DeployFeedbackButton() {
  const newNavigation = useFlag("newNavigation");

  if (newNavigation) {
    return null;
  }

  return (
    <Button variant="outline" size="md" asChild>
      <a href={USERJOT_FEEDBACK_URL} target="_blank" rel="noreferrer">
        <Chats className="size-4" />
        Feedback
      </a>
    </Button>
  );
}
