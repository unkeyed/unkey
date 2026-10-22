import { clickhouse } from "@/lib/clickhouse";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { requestDetailsRequestSchema } from "@unkey/clickhouse/src/frontline";
import { gatewayRequestVerification } from "@unkey/clickhouse/src/verifications";
import { z } from "zod";

export const queryRequestKeyVerification = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(requestDetailsRequestSchema.omit({ workspaceId: true }))
  .output(
    z.object({
      verification: gatewayRequestVerification
        .extend({
          key: z
            .object({ name: z.string().nullable(), apiId: z.string(), projectId: z.string() })
            .nullable(),
        })
        .nullable(),
    }),
  )
  .query(async ({ ctx, input }) => {
    const result = await clickhouse.frontline.keyVerification({
      ...input,
      workspaceId: ctx.workspace.id,
    });

    if (result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch key verification.",
      });
    }

    const verification = result.val[0];
    if (!verification) {
      return { verification: null };
    }

    const [key, api] = await Promise.all([
      db.query.keys.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(
            eq(table.id, verification.key_id),
            eq(table.keyAuthId, verification.key_space_id),
            eq(table.workspaceId, ctx.workspace.id),
            isNull(table.deletedAtM),
          ),
        columns: { name: true },
      }),
      db.query.apis.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(
            eq(table.keyAuthId, verification.key_space_id),
            eq(table.workspaceId, ctx.workspace.id),
            isNull(table.deletedAtM),
          ),
        columns: { id: true, projectId: true },
      }),
    ]);

    return {
      verification: {
        ...verification,
        key: key && api ? { name: key.name, apiId: api.id, projectId: api.projectId } : null,
      },
    };
  });
