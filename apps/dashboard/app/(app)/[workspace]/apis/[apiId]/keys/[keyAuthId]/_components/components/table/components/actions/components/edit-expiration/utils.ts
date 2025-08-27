import { getDefaultValues } from "@/app/(app)/[workspace]/apis/[apiId]/_components/create-key/create-key.utils";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";

export const getKeyExpirationDefaults = (keyDetails: KeyDetails) => {
  const defaultExpiration = keyDetails.expires
    ? new Date(keyDetails.expires)
    : getDefaultValues().expiration?.data;

  return {
    expiration: {
      enabled: Boolean(keyDetails.expires),
      data: defaultExpiration,
    },
  };
};
