import { getDefaultValues } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.utils";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";

export const getKeyExpirationDefaults = (keyDetails: KeyDetails) => {
  const defaultValues = getDefaultValues();
  const defaultExpiration = keyDetails.expires
    ? new Date(keyDetails.expires)
    : defaultValues.expiration?.enabled
      ? defaultValues.expiration.data
      : undefined;

  return {
    expiration: keyDetails.expires
      ? ({
          enabled: true as const,
          data: defaultExpiration,
        } as const)
      : ({
          enabled: false as const,
        } as const),
  };
};
