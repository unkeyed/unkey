import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { cn } from "@/lib/utils";
import { STATUS_STYLES } from "@unkey/ui";

export { STATUS_STYLES };

export const getRowClassName = (log: RootKey, selectedRow: RootKey | null) => {
  const style = STATUS_STYLES;
  const isSelected = log.id === selectedRow?.id;

  return cn(
    style.base,
    style.hover,
    "group rounded",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
    isSelected && style.selected,
  );
};
