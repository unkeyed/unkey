import { z } from "zod";

export const keysListParams = z.object({
  keyAuthId: z.string(),
  names: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      })
    )
    .nullable(),
  identities: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      })
    )
    .nullable(),
  keyIds: z
    .array(
      z.object({
        operator: z.enum(["is", "contains"]),
        value: z.string(),
      })
    )
    .nullable(),
  cursorKeyId: z.string().nullable(),
});

export type KeysListParams = z.infer<typeof keysListParams>;

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
});
export type KeyDetails = z.infer<typeof keyDetailsResponseSchema>;
