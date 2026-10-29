import { Badge } from "@unkey/ui";

export default function BadgeSizes() {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Badge variant="success">Default</Badge>
      <Badge variant="success" size="sm">
        Small
      </Badge>
      <Badge variant="success" size="compact">
        <span className="size-1.5 rounded-full bg-success-9" />
        Compact
      </Badge>
    </div>
  );
}
