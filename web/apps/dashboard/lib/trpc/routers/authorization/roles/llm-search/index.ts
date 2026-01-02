import { env } from "@/lib/env";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import OpenAI from "openai";
import { z } from "zod";
import { getStructuredSearchFromLLM } from "./utils";

const openai = env().OPENAI_API_KEY
  ? new OpenAI({
      apiKey: env().OPENAI_API_KEY,
    })
  : null;

export const rolesLlmSearch = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ query: z.string() }))
  .mutation(async ({ input }) => {
    return await getStructuredSearchFromLLM(openai, input.query);
  });
