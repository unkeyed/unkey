import { env } from "@/lib/env";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import OpenAI from "openai";
import { z } from "zod";
import { getStructuredAuditSearchFromLLM } from "./utils";

const openai = env().OPENAI_API_KEY
  ? new OpenAI({
      apiKey: env().OPENAI_API_KEY,
    })
  : null;

export const auditLogsSearch = rateLimitedProcedure(ratelimit.update)
  .input(z.object({ query: z.string(), timestamp: z.number() }))
  .mutation(async ({ ctx, input }) => {
    return await getStructuredAuditSearchFromLLM(openai, input.query, input.timestamp);
  });
