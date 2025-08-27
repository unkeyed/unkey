import { getDefaultValues } from "@/app/(app)/[workspaceId]/apis/[apiId]/_components/create-key/create-key.utils";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";

export const getKeyRatelimitsDefaults = (keyDetails: KeyDetails) => {
  const defaultRatelimits =
    keyDetails.key.ratelimits.items.length > 0
      ? keyDetails.key.ratelimits.items
      : (getDefaultValues().ratelimit?.data ?? [
          {
            name: "Default",
            limit: 10,
            refillInterval: 1000,
          },
        ]);

  return {
    ratelimit: {
      enabled: keyDetails.key.ratelimits.enabled,
      data: defaultRatelimits,
    },
  };
};
