import { db, schema } from "@unkey/db";
import { z } from "zod";

import { t, auth } from "../trpc";
import { newId } from "@unkey/id";

export const workspaceRouter = t.router({
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        tenantId: z.string(),
        name: z.string().min(1).max(50),
        slug: z
          .string()
          .min(1)
          .max(50)
          .regex(/^[a-zA-Z0-9-_\.]+$/),
      })
    )
    .mutation(async ({ input }) => {
      const id = newId("workspace");
      await db.insert(schema.workspaces).values({
        id,
        tenantId: input.tenantId,
        name: input.name,
        slug: input.slug,
      });
      return {
        id,
      };
    }),
});
