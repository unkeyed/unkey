import { z } from "zod";

export const identitySchema = z.object({
  external_id: z.string().nullable(),
});

export type Identity = z.infer<typeof identitySchema>;

export const ratelimitItemSchema = z.object({
  name: z.string(),
  limit: z.number(),
  refillInterval: z.number(),
  autoApply: z.boolean(),
  id: z.string(),
});

export type RatelimitItem = z.infer<typeof ratelimitItemSchema>;

export const creditsSchema = z.object({
  enabled: z.boolean(),
  remaining: z.number().nullable(),
  refillAmount: z.number().nullable(),
  refillDay: z.number().nullable(),
});

export type Credits = z.infer<typeof creditsSchema>;

export const ratelimitsSchema = z.object({
  enabled: z.boolean(),
  items: z.array(ratelimitItemSchema),
});

export type Ratelimits = z.infer<typeof ratelimitsSchema>;

export const keyDetailsResponseSchema = z.object({
  id: z.string(),
  name: z.string().nullable(),
  owner_id: z.string().nullable(),
  identity_id: z.string().nullable(),
  enabled: z.boolean(),
  expires: z.number().nullable(),
  identity: identitySchema.nullable(),
  updated_at_m: z.number().nullable(),
  metadata: z.string().nullable(),
  start: z.string(),
  key: z.object({
    credits: creditsSchema,
    ratelimits: ratelimitsSchema,
  }),
});

export type KeyDetails = z.infer<typeof keyDetailsResponseSchema>;
