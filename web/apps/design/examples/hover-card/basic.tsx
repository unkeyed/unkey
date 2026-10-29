import { Button, HoverCard, HoverCardContent, HoverCardTrigger } from "@unkey/ui";

export default function HoverCardBasic() {
  return (
    <HoverCard>
      <HoverCardTrigger delay={0} render={<Button variant="outline" />}>
        Hover me
      </HoverCardTrigger>
      <HoverCardContent side="top" className="text-xs">
        <p className="font-medium">checkout-api</p>
        <p className="text-gray-10">Deployed 2 hours ago from main</p>
      </HoverCardContent>
    </HoverCard>
  );
}
