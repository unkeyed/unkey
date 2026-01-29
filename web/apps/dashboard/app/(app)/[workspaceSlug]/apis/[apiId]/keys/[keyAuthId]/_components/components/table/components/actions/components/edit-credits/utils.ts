import { getDefaultValues } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.utils";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import type { z } from "zod";

// biome-ignore format: the comma after z.infer is incorrect syntax
type Refill = z.infer<
  typeof import("@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.schema").refillSchema
>;

export const getKeyLimitDefaults = (keyDetails: KeyDetails) => {
  const defaults = getDefaultValues();
  const defaultLimit = defaults.limit;
  const defaultRemaining =
    keyDetails.key.credits.remaining ?? (defaultLimit?.enabled ? defaultLimit.data.remaining : 100);

  let refill: Refill;
  if (keyDetails.key.credits.refillDay) {
    // Monthly refill
    refill = {
      interval: "monthly",
      amount: keyDetails.key.credits.refillAmount ?? 100,
      refillDay: keyDetails.key.credits.refillDay,
    };
  } else if (keyDetails.key.credits.refillAmount) {
    // Daily refill
    refill = {
      interval: "daily",
      amount: keyDetails.key.credits.refillAmount,
      refillDay: undefined,
    };
  } else {
    // No refill
    refill = {
      interval: "none",
      amount: undefined,
      refillDay: undefined,
    };
  }

  // Return with proper discriminated union types
  return {
    limit: keyDetails.key.credits.enabled
      ? ({
          enabled: true as const,
          data: {
            remaining: defaultRemaining,
            refill,
          },
        } as const)
      : ({
          enabled: false as const,
        } as const),
  };
};
