import { db } from "@/lib/db";
import { searchClient } from "@/lib/search/client";
import { createSearch } from "@/lib/search/engine";
import { ratelimitSearchSpec } from "@/lib/search/specs/ratelimit";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const search = createSearch(ratelimitSearchSpec);

export const ratelimitLlmSearch = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(z.object({ query: z.string(), timestamp: z.number() }))
  .mutation(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
        columns: { id: true },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to verify workspace access. Please try again or contact support@unkey.com if this persists.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.com.",
      });
    }

    return await search(searchClient(), input.query, input.timestamp);
  });
