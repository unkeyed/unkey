import { keysListFilterOperatorEnum } from "@/app/(app)/apis/[apiId]/keys_v2/[keyAuthId]/_components/filters.schema";
import { z } from "zod";

export const keysListParams = z.object({
  keyAuthId: z.string(),
  names: z
    .array(
      z.object({
        operator: keysListFilterOperatorEnum,
        value: z.string(),
      }),
    )
    .nullish(),
  identities: z
    .array(
      z.object({
        operator: keysListFilterOperatorEnum,
        value: z.string(),
      }),
    )
    .nullish(),
  keyIds: z
    .array(
      z.object({
        operator: keysListFilterOperatorEnum,
        value: z.string(),
      }),
    )
    .nullish(),
  cursorKeyId: z.string().nullish(),
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
  key: z.object({
    remaining: z.number().nullable(),
    refillAmount: z.number().nullable(),
  }),
});

export type KeyDetails = z.infer<typeof keyDetailsResponseSchema>;
