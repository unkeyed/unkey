import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { cn } from "@/lib/utils";
import { STATUS_STYLES } from "@unkey/ui";

export { STATUS_STYLES };

export const getRowClassName = (role: RoleBasic, selectedRole: RoleBasic | null) => {
  const style = STATUS_STYLES;
  const isSelected = role.roleId === selectedRole?.roleId;

  return cn(
    style.base,
    style.hover,
    "group rounded-sm",
    "focus:outline-hidden focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
    isSelected && style.selected,
  );
};
