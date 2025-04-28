import {
  type FormValues,
  creditsSchema,
  expirationSchema,
  generalSchema,
  metadataSchema,
  ratelimitSchema,
} from "./schema";
import type { ProcessedFormData, SectionName } from "./types";

/**
 * Processes form data to create the final API payload
 */
export const processFormData = (data: FormValues) => {
  const processedData: ProcessedFormData = {
    prefix: data.prefix || undefined,
    bytes: data.bytes,
    ownerId: data.ownerId?.trim() || undefined,
    name: data.name?.trim() || undefined,
    environment: data.environment || undefined,
    enabled: true,
  };

  // Handle metadata
  if (data.metadata?.enabled && data.metadata.data) {
    try {
      processedData.meta = JSON.parse(data.metadata.data);
    } catch (error) {
      console.error("Failed to parse metadata JSON:", error);
    }
  }

  // Handle limits and refill
  if (data.limit?.enabled && data.limit.data) {
    processedData.remaining = data.limit.data.remaining;

    // Only include refill if interval is not 'none'
    if (data.limit.data.refill?.interval !== "none" && data.limit.data.refill?.amount) {
      processedData.refill = {
        amount: data.limit.data.refill.amount,
        refillDay:
          data.limit.data.refill.interval === "monthly"
            ? data.limit.data.refill.refillDay || 1
            : null,
      };
    }
  }

  // Handle expiration
  if (data.expiration?.enabled && data.expiration.data) {
    processedData.expires = data.expiration.data.getTime();
  }

  // Handle rate limiting
  if (data.ratelimit?.enabled && data.ratelimit.data) {
    processedData.ratelimit = data.ratelimit.data;
  }

  return processedData;
};

export const getDefaultValues = (): Partial<FormValues> => {
  return {
    bytes: 16,
    prefix: "",
    metadata: {
      enabled: false,
    },
    limit: {
      enabled: false,
      data: {
        remaining: 100,
        refill: {
          amount: undefined,
          refillDay: undefined,
        },
      },
    },
    ratelimit: {
      enabled: false,
      data: {
        limit: 10,
        refillInterval: 1000,
      },
    },
    expiration: {
      enabled: false,
    },
  };
};

export const isFeatureEnabled = (sectionId: SectionName, values: FormValues): boolean => {
  switch (sectionId) {
    case "metadata":
      return values.metadata?.enabled || false;
    case "ratelimit":
      return values.ratelimit?.enabled || false;
    case "credits":
      return values.limit?.enabled || false;
    case "expiration":
      return values.expiration?.enabled || false;
    case "general":
      return true;
    default:
      return false;
  }
};

export const sectionSchemaMap = {
  general: generalSchema,
  ratelimit: ratelimitSchema,
  credits: creditsSchema,
  expiration: expirationSchema,
  metadata: metadataSchema,
};

export const getFieldsFromSchema = (schema: any, prefix = ""): string[] => {
  if (!schema?.shape) {
    return [];
  }

  return Object.keys(schema.shape).flatMap((key) => {
    const fullPath = prefix ? `${prefix}.${key}` : key;
    // Handle nested objects recursively
    if (schema.shape[key]?._def?.typeName === "ZodObject") {
      return getFieldsFromSchema(schema.shape[key], fullPath);
    }
    return [fullPath];
  });
};
