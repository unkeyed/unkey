import { Badge } from "@unkey/ui";

export default function BadgeVariants() {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Badge>Primary</Badge>
      <Badge variant="secondary">Secondary</Badge>
      <Badge variant="success">Ready</Badge>
      <Badge variant="warning">Building</Badge>
      <Badge variant="error">Failed</Badge>
      <Badge variant="count">3</Badge>
      <Badge variant="code">acme-api</Badge>
    </div>
  );
}
