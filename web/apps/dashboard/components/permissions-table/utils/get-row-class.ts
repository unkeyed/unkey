import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { cn } from "@/lib/utils";
import { STATUS_STYLES } from "@unkey/ui";

export { STATUS_STYLES };

export const getRowClassName = (log: Permission, selectedRow: Permission | null) => {
  const style = STATUS_STYLES;
  const isSelected = log.permissionId === selectedRow?.permissionId;

  return cn(
    style.base,
    style.hover,
    "group rounded-sm",
    "focus:outline-hidden focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
    isSelected && style.selected,
  );
};
