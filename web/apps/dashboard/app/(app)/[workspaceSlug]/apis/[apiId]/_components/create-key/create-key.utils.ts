import { metadataSchema } from "@/lib/schemas/metadata";
import { deepMerge } from "@/lib/utils";
import type { V2KeysCreateKeyRequestBody } from "@unkey/api/models/components";
import {
  type FormValues,
  creditsSchema,
  expirationSchema,
  generalSchema,
  ratelimitSchema,
} from "./create-key.schema";
import type { SectionName } from "./types";

export const formValuesToCreateKeyRequest = (
  formValues: FormValues,
  apiId: string,
  options: { recoverable?: boolean } = {},
): V2KeysCreateKeyRequestBody => {
  const request: V2KeysCreateKeyRequestBody = {
    apiId,
    byteLength: formValues.bytes,
    enabled: true,
    recoverable: options.recoverable ?? false,
  };

  if (formValues.prefix) {
    request.prefix = formValues.prefix;
  }
  if (formValues.name) {
    request.name = formValues.name;
  }
  if (formValues.externalId) {
    request.externalId = formValues.externalId;
  }
  if (formValues.metadata?.enabled && formValues.metadata.data) {
    request.meta = JSON.parse(formValues.metadata.data);
  }
  if (formValues.expiration?.enabled && formValues.expiration.data) {
    request.expires = formValues.expiration.data.getTime();
  }
  if (formValues.limit?.enabled) {
    request.credits = {
      remaining: formValues.limit.data?.remaining ?? null,
    };
    const refill = formValues.limit.data?.refill;
    if (refill && refill.interval !== "none" && refill.amount) {
      request.credits.refill = {
        interval: refill.interval,
        amount: refill.amount,
        ...(refill.interval === "monthly" && refill.refillDay
          ? { refillDay: refill.refillDay }
          : {}),
      };
    }
  }
  if (formValues.ratelimit?.enabled) {
    request.ratelimits = formValues.ratelimit.data.map((ratelimit) => ({
      name: ratelimit.name,
      limit: ratelimit.limit,
      duration: ratelimit.refillInterval,
      autoApply: ratelimit.autoApply,
    }));
  }

  return request;
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

  const schemaObj = schema as {
    shape: Record<string, unknown>;
    _def?: { typeName: string };
  };

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

export const getDefaultValues = (overrides?: Partial<FormValues> | null): Partial<FormValues> => {
  const defaults: Partial<FormValues> = {
    bytes: 16,
    prefix: "",
    externalId: null,
    identityId: null,
    enabled: true,
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
          autoApply: true,
        },
      ],
    },
    expiration: {
      enabled: false,
    },
  };

  if (!overrides) {
    return defaults;
  }

  return deepMerge(
    defaults as Record<string, unknown>,
    overrides as Record<string, unknown>,
  ) as Partial<FormValues>;
};
