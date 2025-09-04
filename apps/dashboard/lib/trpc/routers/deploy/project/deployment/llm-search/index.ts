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

export const deploymentListLlmSearch = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withLlmAccess())
  .input(z.object({ query: z.string() }))
  .mutation(async ({ ctx }) => {
    return await getStructuredSearchFromLLM(openai, ctx.validatedQuery);
  });
