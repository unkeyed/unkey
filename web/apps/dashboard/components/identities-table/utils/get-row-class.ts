import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { cn } from "@/lib/utils";
import type { z } from "zod";

type Identity = z.infer<typeof IdentityResponseSchema>;

export function getRowClassName(identity: Identity, selectedIdentity: Identity | null): string {
  return cn(
    "hover:bg-gray-2 transition-colors cursor-pointer",
    selectedIdentity?.id === identity.id && "bg-gray-3",
  );
}
