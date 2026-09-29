import { Badge } from "@unkey/ui";

function getMethodVariant(method: string): "success" | "warning" | "error" | "primary" | "blocked" {
  switch (method) {
    case "GET":
      return "success";
    case "POST":
      return "warning";
    default:
      return "primary";
  }
}

export const MethodBadge: React.FC<{ method: string }> = ({ method }) => (
  <Badge
    variant={getMethodVariant(method)}
    size="sm"
    className="text-[11px] font-medium w-10 h-4.5 flex items-center justify-center"
  >
    {method}
  </Badge>
);
