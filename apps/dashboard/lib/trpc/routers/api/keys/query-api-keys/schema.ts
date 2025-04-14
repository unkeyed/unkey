import { z } from "zod";

export const identitySchema = z.object({
  external_id: z.string().nullable(),
});

export type Identity = z.infer<typeof identitySchema>;

export const keyDetailsResponseSchema = z.object({
  id: z.string(),
  name: z.string().nullable(),
  owner_id: z.string().nullable(),
  identity_id: z.string().nullable(),
  enabled: z.boolean(),
  expires: z.number().nullable(),
  identity: identitySchema.nullable(),
  updated_at_m: z.number().nullable(),
  start: z.string(),
  key: z.object({
    remaining: z.number().nullable(),
    refillAmount: z.number().nullable(),
  }),
});

export type KeyDetails = z.infer<typeof keyDetailsResponseSchema>;
