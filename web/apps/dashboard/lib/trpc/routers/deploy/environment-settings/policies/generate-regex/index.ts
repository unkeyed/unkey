import { env } from "@/lib/env";
import { withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import OpenAI from "openai";
import { z } from "zod";
import { generateRegexFromLLM } from "./utils";

const openai = env().OPENAI_API_KEY
  ? new OpenAI({
      apiKey: env().OPENAI_API_KEY,
    })
  : null;

export const generateRegex = workspaceProcedure
  .use(withLlmAccess())
  .input(
    z.object({
      query: z.string(),
      conditionType: z.enum(["path", "header", "queryParam"]),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    return await generateRegexFromLLM(openai, ctx.validatedQuery, input.conditionType);
  });
