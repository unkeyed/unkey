import { z } from "zod";
import {
  namespaceFilterOperatorEnum,
  namespaceListFilterFieldNames,
} from "../namespace-list-filters.schema";

const filterItemSchema = z.object({
  operator: namespaceFilterOperatorEnum,
  value: z.string().trim().max(60, "Query too long"),
});

const cursor = z.object({
  id: z.string(),
});

export type CursorType = z.infer<typeof cursor>;

// Build filter fields with proper typing
const filterFields = namespaceListFilterFieldNames.reduce(
  (acc, fieldName) => {
    acc[fieldName] = z.array(filterItemSchema).nullish();
    return acc;
  },
  {} as Record<
    (typeof namespaceListFilterFieldNames)[number],
    z.ZodOptional<z.ZodNullable<z.ZodArray<typeof filterItemSchema>>>
  >,
);

export const namespaceListInputSchema = z
  .object({
    cursor: cursor.optional(),
  })
  .extend(z.object(filterFields).shape);

export type NamespaceListInputSchema = z.infer<typeof namespaceListInputSchema>;

export const ratelimitNamespace = z.object({
  id: z.string(),
  name: z.string(),
});

export type RatelimitNamespace = z.infer<typeof ratelimitNamespace>;

export const namespaceListOutputSchema = z.object({
  namespaceList: z.array(ratelimitNamespace),
  hasMore: z.boolean(),
  nextCursor: cursor.optional(),
  total: z.number(),
});

export type NamespaceListOutputSchema = z.infer<typeof namespaceListOutputSchema>;
