import { z } from "zod";

export const keysQueryListPayload = z.object({
  keyAuthId: z.string(),
  cursor: z
    .object({
      keyId: z.string().nullable(),
    })
    .optional()
    .nullable(),
  keyIds: z
    .array(
      z.object({
        operator: z.enum(["is", "contains"]),
        value: z.string(),
      }),
    )
    .optional()
    .nullable(),
  names: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      }),
    )
    .optional()
    .nullable(),
  identities: z
    .array(
      z.object({
        operator: z.enum(["is", "contains", "startsWith", "endsWith"]),
        value: z.string(),
      }),
    )
    .optional()
    .nullable(),
});

export type KeysQueryListPayload = z.infer<typeof keysQueryListPayload>;
