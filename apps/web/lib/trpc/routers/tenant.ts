import { db, schema } from "@unkey/db";
import { z } from "zod";

import { t, auth } from "../trpc";

export const tenantRouter = t.router({
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        tenantId: z.string(),
        name: z.string().min(1).max(50),
        slug: z.string().min(1).max(50).regex(/^[a-zA-Z0-9-_\.]+$/),
      }),
    )
    .mutation(async ({ input }) => {
      const id = input.tenantId;
      await db.insert(schema.tenants).values({
        id,
        name: input.name,
        slug: input.slug,
      });
      return {
        id,
      };
    }),
});
