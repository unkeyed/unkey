import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { cn } from "@/lib/utils";
import { STATUS_STYLES } from "@unkey/ui";

export const getRowClassName = (permission: Permission, selectedPermission: Permission | null) => {
  const isSelected = permission.permissionId === selectedPermission?.permissionId;

  return cn(
    STATUS_STYLES.base,
    STATUS_STYLES.hover,
    "group rounded-sm",
    "focus:outline-hidden focus:ring-1 focus:ring-opacity-40",
    STATUS_STYLES.focusRing,
    isSelected && STATUS_STYLES.selected,
  );
};
