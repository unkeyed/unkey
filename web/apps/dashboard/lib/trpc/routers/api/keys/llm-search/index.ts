import { db } from "@/lib/db";
import { searchClient } from "@/lib/search/client";
import { createSearch } from "@/lib/search/engine";
import { keysOverviewSearchSpec } from "@/lib/search/specs/keys-overview";
import { withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const search = createSearch(keysOverviewSearchSpec);

export const keysLlmSearch = workspaceProcedure
  .use(withLlmAccess())
  .input(
    z.object({
      query: z.string(),
      timestamp: z.number(),
      apiId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    // Verify API access and workspace permissions
    const api = await db.query.apis
      .findFirst({
        where: (api, { and, eq, isNull }) =>
          and(
            eq(api.id, input.apiId),
            eq(api.workspaceId, ctx.workspace.id),
            isNull(api.deletedAtM),
          ),
        columns: { id: true },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to verify API access. Please try again or contact support@unkey.com if this persists.",
        });
      });

    if (!api) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "API not found or you don't have access to it.",
      });
    }

    // Process the natural language query using LLM
    return await search(searchClient(), input.query, input.timestamp);
  });
