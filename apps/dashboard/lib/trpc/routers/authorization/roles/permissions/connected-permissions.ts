import { and, count, db, eq } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { rolesPermissions } from "@unkey/db/src/schema";
import { z } from "zod";

const permissionsResponse = z.object({
  totalCount: z.number(),
});

export const queryRolePermissions = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(
    z.object({
      roleId: z.string(),
    }),
  )
  .output(permissionsResponse)
  .query(async ({ ctx, input }) => {
    const workspaceId = ctx.workspace.id;
    const { roleId } = input;

    const result = await db
      .select({ count: count() })
      .from(rolesPermissions)
      .where(
        and(eq(rolesPermissions.workspaceId, workspaceId), eq(rolesPermissions.roleId, roleId)),
      );

    return { totalCount: result?.[0]?.count ?? 0 };
  });
