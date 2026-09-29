import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

// The identity detail page reads the public API SDK, whose Identity model has
// no projectId. This procedure is the dashboard-only source for it.
const identityDetailsInput = z.object({
  identityId: z.string().min(1),
});

const identityDetailsOutput = z.object({
  id: z.string(),
  projectId: z.string(),
});

export const queryIdentityDetails = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(identityDetailsInput)
  .output(identityDetailsOutput)
  .query(async ({ ctx, input }) => {
    const identity = await db.query.identities
      .findFirst({
        where: (table, { eq, and }) =>
          and(eq(table.id, input.identityId), eq(table.workspaceId, ctx.workspace.id)),
        columns: { id: true, projectId: true },
      })
      .catch((error) => {
        console.error("Failed to fetch identity details", {
          identityId: input.identityId,
          workspaceId: ctx.workspace.id,
          error: error instanceof Error ? error.message : error,
        });
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Failed to fetch identity details",
        });
      });

    if (!identity) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Identity not found",
      });
    }

    return identity;
  });
