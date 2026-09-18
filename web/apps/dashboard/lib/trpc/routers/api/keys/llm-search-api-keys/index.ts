import { db } from "@/lib/db";
import { searchClient } from "@/lib/search/client";
import { createSearch } from "@/lib/search/engine";
import { keysListSearchSpec } from "@/lib/search/specs/keys-list";
import { withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const search = createSearch(keysListSearchSpec);

export const apiKeysLlmSearch = workspaceProcedure
  .use(withLlmAccess())
  .input(
    z.object({
      query: z.string(),
      keyspaceId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    // Verify API access and workspace permissions
    const api = await db.query.apis
      .findFirst({
        where: (api, { and, eq, isNull }) =>
          and(
            eq(api.keyAuthId, input.keyspaceId),
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

    // Process the natural language query using LLM. Use ctx.validatedQuery from
    // withLlmAccess(), which enforces 3-120 char length and the LLM rate limit.
    return await search(searchClient(), ctx.validatedQuery);
  });
