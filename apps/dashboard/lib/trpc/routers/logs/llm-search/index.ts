import { env } from "@/lib/env";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import OpenAI from "openai";
import { z } from "zod";
import { getStructuredSearchFromLLM } from "./utils";

const openai = env().OPENAI_API_KEY
  ? new OpenAI({
      apiKey: env().OPENAI_API_KEY,
    })
  : null;

export const llmSearch = rateLimitedProcedure(ratelimit.update)
  .input(z.object({ query: z.string(), timestamp: z.number() }))
  .mutation(async ({ input }) => {
    return await getStructuredSearchFromLLM(openai, input.query, input.timestamp);
  });
