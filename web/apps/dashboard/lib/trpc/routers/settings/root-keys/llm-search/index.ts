import OpenAI from "openai";
import { z } from "zod";
import { env } from "@/lib/env";
import { withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import { getStructuredSearchFromLLM } from "./utils";

const openai = env().OPENAI_API_KEY
  ? new OpenAI({
      apiKey: env().OPENAI_API_KEY,
    })
  : null;

export const rootKeysLlmSearch = workspaceProcedure
  .use(withLlmAccess())
  .input(z.object({ query: z.string() }))
  .mutation(async ({ input }) => {
    return await getStructuredSearchFromLLM(openai, input.query);
  });
