import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";

export const getKeyMetadataDefaults = (keyDetails: KeyDetails) => {
  return {
    metadata: {
      enabled: Boolean(keyDetails.metadata),
      data: JSON.stringify(JSON.parse(keyDetails.metadata || "{}"), null, 2) ?? undefined,
    },
  };
};
