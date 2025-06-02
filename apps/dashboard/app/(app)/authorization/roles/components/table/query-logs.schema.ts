import { z } from "zod";
import { rolesFilterOperatorEnum } from "../../filters.schema";

export const queryRolesPayload = z.object({
  limit: z.number().int(),
  slug: z
    .object({
      filters: z.array(
        z.object({
          operator: rolesFilterOperatorEnum,
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  description: z
    .object({
      filters: z.array(
        z.object({
          operator: rolesFilterOperatorEnum,
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  name: z
    .object({
      filters: z.array(
        z.object({
          operator: rolesFilterOperatorEnum,
          value: z.string(),
        }),
      ),
    })
    .nullable(),
  cursor: z.number().nullable().optional().nullable(),
});
