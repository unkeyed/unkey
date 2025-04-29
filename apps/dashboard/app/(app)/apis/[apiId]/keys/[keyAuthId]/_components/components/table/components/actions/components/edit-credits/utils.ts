import type { refillSchema } from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.schema";
import { getDefaultValues } from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.utils";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import type { z } from "zod";

type Refill = z.infer<typeof refillSchema>;
export const getKeyLimitDefaults = (keyDetails: KeyDetails) => {
  const hasRemaining = Boolean(keyDetails.key.remaining);
  const defaultRemaining =
    keyDetails.key.remaining ?? getDefaultValues().limit?.data?.remaining ?? 100;

  let refill: Refill;
  if (keyDetails.key.refillDay) {
    // Monthly refill
    refill = {
      interval: "monthly",
      amount: keyDetails.key.refillAmount ?? 100,
      refillDay: keyDetails.key.refillDay,
    };
  } else if (keyDetails.key.refillAmount) {
    // Daily refill
    refill = {
      interval: "daily",
      amount: keyDetails.key.refillAmount,
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
      enabled: hasRemaining,
      data: {
        remaining: defaultRemaining,
        refill,
      },
    },
  };
};
