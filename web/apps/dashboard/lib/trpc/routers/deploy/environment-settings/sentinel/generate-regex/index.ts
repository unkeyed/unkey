import { openai } from "@/lib/openai";
import { withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { generateRegexFromLLM } from "./utils";

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
