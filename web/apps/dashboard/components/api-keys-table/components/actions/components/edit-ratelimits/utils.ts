import { getDefaultValues } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.utils";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";

export const getKeyRatelimitsDefaults = (keyDetails: KeyDetails) => {
  const defaultValues = getDefaultValues();
  const defaultRatelimits =
    keyDetails.key.ratelimits.items.length > 0
      ? keyDetails.key.ratelimits.items
      : defaultValues.ratelimit?.enabled
        ? defaultValues.ratelimit.data
        : [
            {
              name: "Default",
              limit: 10,
              refillInterval: 1000,
            },
          ];

  return {
    ratelimit: keyDetails.key.ratelimits.enabled
      ? ({
          enabled: true as const,
          data: defaultRatelimits,
        } as const)
      : ({
          enabled: false as const,
        } as const),
  };
};
