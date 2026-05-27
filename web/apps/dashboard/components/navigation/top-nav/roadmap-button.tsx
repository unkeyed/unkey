import { Bolt } from "@unkey/icons";
import { Button } from "@unkey/ui";

const USERJOT_ROADMAP_URL = "https://unkey.userjot.com/roadmap";

export function TopNavRoadmapButton({ className }: { className?: string }) {
  return (
    <Button variant="outline" size="sm" asChild className={className}>
      <a href={USERJOT_ROADMAP_URL} target="_blank" rel="noreferrer">
        <Bolt className="size-4" />
        Roadmap
      </a>
    </Button>
  );
}
