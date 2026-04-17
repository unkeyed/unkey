import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { cn } from "@/lib/utils";
import { STATUS_STYLES } from "@unkey/ui";
import type { z } from "zod";

type Identity = z.infer<typeof IdentityResponseSchema>;

export { STATUS_STYLES };

export const getRowClassName = (identity: Identity, selectedIdentity: Identity | null): string => {
  const style = STATUS_STYLES;
  const isSelected = identity.id === selectedIdentity?.id;

  return cn(
    style.base,
    style.hover,
    "group rounded",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    style.focusRing,
    isSelected && style.selected,
  );
};
