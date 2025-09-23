import { getDefaultValues } from "@/app/(app)/[workspace]/apis/[apiId]/_components/create-key/create-key.utils";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import type { z } from "zod";

// biome-ignore format: the comma after z.infer is incorrect syntax
type Refill = z.infer<
  typeof import("@/app/(app)/[workspace]/apis/[apiId]/_components/create-key/create-key.schema").refillSchema
>;

export const getKeyLimitDefaults = (keyDetails: KeyDetails) => {
  const defaultRemaining =
    keyDetails.key.credits.remaining ?? getDefaultValues().limit?.data?.remaining ?? 100;

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

  return {
    limit: {
      enabled: keyDetails.key.credits.enabled,
      data: {
        remaining: defaultRemaining,
        refill,
      },
    },
  };
};
