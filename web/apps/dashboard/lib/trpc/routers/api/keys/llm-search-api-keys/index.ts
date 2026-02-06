import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import OpenAI from "openai";
import { z } from "zod";
import { getKeysStructuredSearchFromLLM } from "./utils";

const openai = env().OPENAI_API_KEY
  ? new OpenAI({
      apiKey: env().OPENAI_API_KEY,
    })
  : null;

export const apiKeysLlmSearch = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
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
    return await getKeysStructuredSearchFromLLM(openai, input.query);
  });
