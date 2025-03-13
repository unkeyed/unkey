import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { TRPCError } from "@trpc/server";
import OpenAI from "openai";
import { z } from "zod";
import { getKeysStructuredSearchFromLLM } from "./utils";

const openai = env().OPENAI_API_KEY
  ? new OpenAI({
      apiKey: env().OPENAI_API_KEY,
    })
  : null;

export const keysLlmSearch = rateLimitedProcedure(ratelimit.update)
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
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to verify API access. Please try again or contact support@unkey.dev if this persists.",
        });
      });

    if (!api) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "API not found or you don't have access to it.",
      });
    }

    // Process the natural language query using LLM
    return await getKeysStructuredSearchFromLLM(openai, input.query, input.timestamp);
  });
