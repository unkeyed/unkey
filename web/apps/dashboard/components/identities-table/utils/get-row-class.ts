import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { cn } from "@/lib/utils";
import { STATUS_STYLES } from "@unkey/ui";
import type { z } from "zod";

type Identity = z.infer<typeof IdentityResponseSchema>;

export const getRowClassName = (identity: Identity, selectedIdentity: Identity | null): string => {
  const isSelected = identity.id === selectedIdentity?.id;

  return cn(
    STATUS_STYLES.base,
    STATUS_STYLES.hover,
    "group rounded",
    "focus:outline-none focus:ring-1 focus:ring-opacity-40",
    STATUS_STYLES.focusRing,
    isSelected && STATUS_STYLES.selected,
  );
};
