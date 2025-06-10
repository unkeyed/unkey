import { env } from "@/lib/env";
import { requireUser, requireWorkspace, t, withLlmAccess } from "@/lib/trpc/trpc";
import OpenAI from "openai";
import { z } from "zod";
import { getStructuredSearchFromLLM } from "./utils";

const openai = env().OPENAI_API_KEY
  ? new OpenAI({
      apiKey: env().OPENAI_API_KEY,
    })
  : null;

export const llmSearch = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withLlmAccess())
  .input(z.object({ timestamp: z.number() }))
  .mutation(async ({ input, ctx }) => {
    return await getStructuredSearchFromLLM(openai, ctx.validatedQuery, input.timestamp);
  });
