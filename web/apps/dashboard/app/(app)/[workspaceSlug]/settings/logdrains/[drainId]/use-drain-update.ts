"use client";

import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { inferRouterInputs } from "@trpc/server";
import { toast } from "@unkey/ui";

type UpdateInput = inferRouterInputs<Router>["logdrain"]["update"];

export function useDrainUpdate(drainId: string, onSuccess?: (variables: UpdateInput) => void) {
  const utils = trpc.useUtils();
  return trpc.logdrain.update.useMutation({
    onSuccess: (_data, variables) => {
      onSuccess?.(variables);
      utils.logdrain.list.invalidate();
      utils.logdrain.get.invalidate({ id: drainId });
      toast.success("Log drain updated");
    },
    onError: (error) => toast.error(error.message),
  });
}
