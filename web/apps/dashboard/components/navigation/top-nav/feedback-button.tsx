import { Chats } from "@unkey/icons";
import { Button } from "@unkey/ui";

const USERJOT_FEEDBACK_URL = "https://feedback.unkey.com/";

export function TopNavFeedbackButton({ className }: { className?: string }) {
  return (
    <Button variant="outline" size="sm" asChild className={className}>
      <a href={USERJOT_FEEDBACK_URL} target="_blank" rel="noreferrer">
        <Chats className="size-4" />
        Feedback
      </a>
    </Button>
  );
}
