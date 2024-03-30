import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../../trpc";

const nameSchema = z
  .string()
  .min(3)
  .regex(/^[a-zA-Z0-9_\-\.\*]+$/, {
    message:
      "Must be at least 3 characters long and only contain alphanumeric, periods, dashes and underscores",
  });

export const updateRole = t.procedure
  .use(auth)
  .input(
    z.object({
      id: z.string(),
      name: nameSchema,
      description: z.string().nullable(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const workspace = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      with: {
        roles: {
          where: (table, { eq }) => eq(table.id, input.id),
        },
      },
    });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }
    if (workspace.roles.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "role not found",
      });
    }
    await db.update(schema.roles).set(input).where(eq(schema.roles.id, input.id));
  });
