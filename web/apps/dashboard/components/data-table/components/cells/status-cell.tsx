import { Check, Clock, XMark } from "@unkey/icons";
import { Badge } from "@unkey/ui";

interface StatusCellProps {
  status: "success" | "pending" | "error" | "warning" | "active" | "inactive";
  label?: string;
  showIcon?: boolean;
}

/**
 * Status cell with badge and optional icon
 */
export function StatusCell({ status, label, showIcon = true }: StatusCellProps) {
  const config = getStatusConfig(status);

  return (
    <Badge variant={config.variant}>
      {showIcon && config.icon && <config.icon className="size-3" />}
      {label || config.label}
    </Badge>
  );
}

function getStatusConfig(status: StatusCellProps["status"]) {
  switch (status) {
    case "success":
    case "active":
      return {
        variant: "success" as const,
        label: status === "success" ? "Success" : "Active",
        icon: Check,
      };
    case "pending":
      return {
        variant: "primary" as const,
        label: "Pending",
        icon: Clock,
      };
    case "warning":
      return {
        variant: "warning" as const,
        label: "Warning",
        icon: Clock,
      };
    case "error":
    case "inactive":
      return {
        variant: "error" as const,
        label: status === "error" ? "Error" : "Inactive",
        icon: XMark,
      };
  }
}
