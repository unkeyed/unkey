import { db, schema } from "@unkey/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { t, auth } from "../trpc";
import { newId } from "@unkey/id";

export const teamRouter = t.router({
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        id: z.string(),
        name: z.string().min(1).max(50),
        slug: z.string().min(1).max(50).regex(/^[a-zA-Z0-9-_\.]+$/),
      }),
    )
    .mutation(async ({ input }) => {
      const id = input.id;
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
