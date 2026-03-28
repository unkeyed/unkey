import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";

export const getKeyMetadataDefaults = (keyDetails: KeyDetails) => {
  const hasMetadata = Boolean(keyDetails.metadata);
  const metadataData = JSON.stringify(JSON.parse(keyDetails.metadata || "{}"), null, 2);

  return {
    metadata: hasMetadata
      ? ({
          enabled: true as const,
          data: metadataData,
        } as const)
      : ({
          enabled: false as const,
        } as const),
  };
};
