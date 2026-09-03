import { cn } from "@/lib/utils";
import type { AlertStatus } from "./types";

export function AlertStatusBadge({ status }: { status: AlertStatus }) {
  return (
    <span className="inline-flex items-center gap-2 whitespace-nowrap text-xs font-medium text-gray-11">
      <span
        className={cn("size-2 rounded-full", status === "open" ? "bg-error-9" : "bg-success-9")}
      />
      {status === "open" ? "Open" : "Resolved"}
    </span>
  );
}
