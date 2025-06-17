import { deepMerge } from "@/lib/utils";
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
    prefix: formValues.prefix === "" ? undefined : formValues.prefix,
    bytes: formValues.bytes,
    externalId: formValues.externalId || null,
    name: formValues.name === "" ? undefined : formValues.name,
    enabled: true,
    environment: formValues.environment === "" ? undefined : formValues.name,
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

export const getFieldsFromSchema = (schema: unknown, prefix = ""): string[] => {
  if (!schema || typeof schema !== "object" || !("shape" in schema)) {
    return [];
  }

  const schemaObj = schema as { shape: Record<string, unknown>; _def?: { typeName: string } };

  return Object.keys(schemaObj.shape).flatMap((key) => {
    const fullPath = prefix ? `${prefix}.${key}` : key;
    const fieldSchema = schemaObj.shape[key];

    if (
      fieldSchema &&
      typeof fieldSchema === "object" &&
      "_def" in fieldSchema &&
      (fieldSchema as { _def: { typeName: string } })._def?.typeName === "ZodObject"
    ) {
      return getFieldsFromSchema(fieldSchema, fullPath);
    }
    return [fullPath];
  });
};

export const getDefaultValues = (
  overrides?: Partial<FormValueTypes> | null,
): Partial<FormValueTypes> => {
  const defaults = {
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
          interval: "none" as const,
          amount: undefined,
          refillDay: undefined,
        },
      },
    },
    ratelimit: {
      enabled: false,
      data: [
        {
          name: "Default",
          limit: 10,
          refillInterval: 1000,
        },
      ],
    },
    expiration: {
      enabled: false,
    },
  };

  // biome-ignore lint/suspicious/noExplicitAny: This will be merged with actual default, so its okay to use any
  return overrides ? deepMerge(defaults, overrides as any) : defaults;
};
