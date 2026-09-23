import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { STATUS_STYLES } from "@unkey/ui";
import { cn } from "cn";

export { STATUS_STYLES };

export const getRowClassName = (key: KeyDetails, selectedKey: KeyDetails | null) => {
  const style = STATUS_STYLES;
  const isSelected = key.id === selectedKey?.id;

  return cn(
    style.base,
    style.hover,
    "group rounded-sm",
    "focus:outline-none focus:ring-1",
    style.focusRing,
    isSelected && style.selected,
  );
};
