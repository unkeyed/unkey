import {
  expirationSchema,
  generalSchema,
  limitSchema,
  metadataSchema,
  ratelimitSchema,
  type FormValues,
} from "./schema";

/**
 * Processes form data to create the final API payload
 */
export const processFormData = (data: FormValues) => {
  const processedData: Record<string, any> = {
    // Include the base fields
    prefix: data.prefix || undefined,
    bytes: data.bytes,
    ownerId: data.ownerId?.trim() || undefined,
    name: data.name?.trim() || undefined,
    environment: data.environment || undefined,
  };

  // Only include enabled features
  if (data.metaEnabled && data.meta) {
    try {
      processedData.meta = JSON.parse(data.meta);
    } catch (error) {
      console.error("Failed to parse metadata JSON:", error);
    }
  }

  if (data.limitEnabled && data.limit) {
    processedData.limit = {
      remaining: data.limit.remaining,
    };

    // Only include refill if interval is not 'none'
    if (data.limit.refill?.interval !== "none" && data.limit.refill?.amount) {
      processedData.limit.refill = {
        interval: data.limit.refill.interval,
        amount: data.limit.refill.amount,
      };

      // Only include refill day for monthly intervals
      if (data.limit.refill.interval === "monthly") {
        processedData.limit.refill.refillDay = data.limit.refill.refillDay || 1;
      }
    }
  }

  if (data.expireEnabled && data.expires) {
    processedData.expires = data.expires.getTime();
  }

  if (data.ratelimitEnabled && data.ratelimit) {
    processedData.ratelimit = {
      duration: data.ratelimit.refillInterval,
      limit: data.ratelimit.limit,
    };
  }

  return processedData;
};

/**
 * Resets sections of the form based on toggles
 */
export const getResetFieldsForSection = (sectionName: string) => {
  switch (sectionName) {
    case "meta":
      return ["meta"];
    case "limit":
      return [
        "limit.remaining",
        "limit.refill.amount",
        "limit.refill.interval",
        "limit.refill.refillDay",
      ];
    case "ratelimit":
      return ["ratelimit.duration", "ratelimit.limit", "ratelimit.async"];
    case "expire":
      return ["expires"];
    default:
      return [];
  }
};

export const getDefaultValues = (): Partial<FormValues> => {
  return {
    bytes: 16,
    prefix: "",
    metaEnabled: false,
    limitEnabled: false,
    ratelimitEnabled: false,
    expireEnabled: false,
    limit: {
      remaining: 0,
      refill: {
        interval: "none",
        amount: undefined,
        refillDay: undefined,
      },
    },
  };
};

export const sectionSchemaMap = {
  general: generalSchema,
  ratelimit: ratelimitSchema,
  "usage-limit": limitSchema,
  expiration: expirationSchema,
  metadata: metadataSchema,
};

export const getFieldsFromSchema = (schema: any, prefix = ""): string[] => {
  if (!schema?.shape) {
    return [];
  }

  return Object.keys(schema.shape).flatMap((key) => {
    const fullPath = prefix ? `${prefix}.${key}` : key;

    if (key === "_sectionValid") {
      return [];
    }

    // Handle nested objects recursively
    if (schema.shape[key]?._def?.typeName === "ZodObject") {
      return getFieldsFromSchema(schema.shape[key], fullPath);
    }

    return [fullPath];
  });
};
