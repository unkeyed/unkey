import {
  type CreateKeyInput,
  type FormValueTypes,
  type FormValues,
  creditsSchema,
  expirationSchema,
  generalSchema,
  metadataSchema,
  ratelimitSchema,
} from "./create-key.schema";
import type { SectionName } from "./types";

/**
 * Processes form data to create the final API payload
 */
export const formValuesToApiInput = (formValues: FormValues, keyAuthId: string): CreateKeyInput => {
  return {
    keyAuthId,
    prefix: formValues.prefix,
    bytes: formValues.bytes,
    ownerId: formValues.ownerId || null,
    name: formValues.name,
    enabled: true,
    environment: formValues.environment,
    meta:
      formValues.metadata?.enabled && formValues.metadata.data
        ? JSON.parse(formValues.metadata.data)
        : undefined,
    remaining: formValues.limit?.enabled ? formValues.limit.data?.remaining : undefined,
    refill:
      formValues.limit?.enabled && formValues.limit.data?.refill?.interval !== "none"
        ? {
            amount: formValues.limit.data?.refill?.amount as number,
            refillDay:
              formValues.limit.data?.refill?.interval === "monthly"
                ? formValues.limit.data?.refill?.refillDay || null
                : null,
          }
        : undefined,
    expires:
      formValues.expiration?.enabled && formValues.expiration.data
        ? formValues.expiration.data.getTime()
        : undefined,
    ratelimit: formValues.ratelimit?.enabled ? formValues.ratelimit.data : undefined,
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

export const getDefaultValues = (): Partial<FormValueTypes> => {
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
      data: [
        {
          name: "default",
          limit: 10,
          refillInterval: 1000,
        },
      ],
    },
    expiration: {
      enabled: false,
    },
  };
};
