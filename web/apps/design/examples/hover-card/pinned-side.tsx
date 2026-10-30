import { Button, HoverCard, HoverCardContent, HoverCardTrigger } from "@unkey/ui";

export default function HoverCardPinnedSide() {
  return (
    <HoverCard>
      <HoverCardTrigger delay={0} render={<Button variant="outline" />}>
        Crash events
      </HoverCardTrigger>
      <HoverCardContent
        side="bottom"
        collisionAvoidance={{ side: "none" }}
        className="w-72 text-xs"
      >
        <p className="font-medium">Crash events 6</p>
        <p className="text-gray-10">Oct 26, 8:00 PM – 8:01 PM</p>
      </HoverCardContent>
    </HoverCard>
  );
}
