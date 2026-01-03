import { z } from "zod";

export const LIMIT = 50;

export const keysQueryPayload = z.object({
  cursor: z.string().optional(),
  limit: z.number().default(LIMIT),
});

export const RoleSchema = z.object({
  id: z.string(),
  name: z.string(),
});

export const KeyResponseSchema = z.object({
  id: z.string(),
  name: z.string().nullable(),
  roles: z.array(RoleSchema),
});

export const KeysResponse = z.object({
  keys: z.array(KeyResponseSchema),
  hasMore: z.boolean(),
  nextCursor: z.string().nullish(),
});

export const keysSearchPayload = z.object({
  query: z.string().min(1, "Search query cannot be empty"),
});

export const KeysSearchResponse = z.object({
  keys: z.array(KeyResponseSchema),
});

type KeyWithRoles = {
  id: string;
  name: string | null;
  roles: {
    role: { id: string; name: string } | null;
  }[];
};

export const transformKey = (key: KeyWithRoles) => ({
  id: key.id,
  name: key.name,
  roles: key.roles
    .filter(
      (
        keyRole,
      ): keyRole is typeof keyRole & {
        role: NonNullable<typeof keyRole.role>;
      } => keyRole.role !== null,
    )
    .map((keyRole) => ({
      id: keyRole.role.id,
      name: keyRole.role.name,
    })),
});
