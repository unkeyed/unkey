import { Badge } from "@unkey/ui";

function getMethodVariant(method: string): "success" | "warning" | "error" | "primary" | "blocked" {
  switch (method) {
    case "GET":
    case "HEAD":
      return "success";
    case "POST":
      return "warning";
    case "PUT":
    case "PATCH":
      return "blocked";
    case "DELETE":
      return "error";
    default:
      return "primary";
  }
}

export const MethodBadge: React.FC<{ method: string }> = ({ method }) => (
  <Badge
    variant={getMethodVariant(method)}
    size="sm"
    className="text-[11px] font-medium w-10 h-[18px] flex items-center justify-center"
  >
    {method}
  </Badge>
);
