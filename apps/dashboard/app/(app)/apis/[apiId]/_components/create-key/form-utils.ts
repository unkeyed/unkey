import {
  type FormValues,
  expirationSchema,
  generalSchema,
  limitSchema,
  metadataSchema,
  ratelimitSchema,
} from "./schema";

/**
 * Processes form data to create the final API payload
 */
export const processFormData = (data: FormValues) => {
  const processedData: Record<string, any> = {
    // Include the base fields
    prefix: data.prefix || null,
    bytes: data.bytes,
    ownerId: data.ownerId?.trim() || null,
    name: data.name?.trim() || null,
    environment: data.environment || null,
  };

  // Only include enabled features
  if (data.metadata?.enabled && data.metadata.data) {
    try {
      processedData.metaEnabled = true;
      processedData.meta = JSON.parse(data.metadata.data);
    } catch (error) {
      console.error("Failed to parse metadata JSON:", error);
    }
  }

  if (data.limit?.enabled && data.limit.data) {
    processedData.limitEnabled = true;
    processedData.limit = {
      remaining: data.limit.data.remaining,
    };

    // Only include refill if interval is not 'none'
    if (data.limit.data.refill?.interval !== "none" && data.limit.data.refill?.amount) {
      processedData.limit.refill = {
        interval: data.limit.data.refill.interval,
        amount: data.limit.data.refill.amount,
      };

      // Only include refill day for monthly intervals
      if (data.limit.data.refill.interval === "monthly") {
        processedData.limit.refill.refillDay = data.limit.data.refill.refillDay || 1;
      }
    }
  }

  if (data.expiration?.enabled && data.expiration.data) {
    processedData.expireEnabled = true;
    processedData.expires = data.expiration.data.getTime();
  }

  if (data.ratelimit?.enabled && data.ratelimit.data) {
    processedData.ratelimitEnabled = true;
    processedData.ratelimit = {
      async: false,
      duration: data.ratelimit.data.refillInterval,
      limit: data.ratelimit.data.limit,
    };
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
          interval: "none",
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

export const sectionSchemaMap = {
  general: generalSchema,
  ratelimit: ratelimitSchema,
  credits: limitSchema,
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
