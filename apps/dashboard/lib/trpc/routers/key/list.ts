import { and, db, eq, isNull, schema } from "@/lib/db";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const listKeys = t.procedure
  .use(auth)
  .input(
    z.object({
      keyAuthId: z.string(),
    }),
  )
  .query(async ({ input }) => {
    const keys = await db.query.keys.findMany({
      where: and(eq(schema.keys.keyAuthId, input.keyAuthId), isNull(schema.keys.deletedAt)),
      limit: 100,
      with: {
        roles: true,
        permissions: true,
      },
    });

    return keys;
  });
