import { openai } from "@/lib/openai";
import { withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { generatePoliciesFromLLM } from "./utils";

export const generatePolicies = workspaceProcedure
  .use(withLlmAccess())
  .input(z.object({ query: z.string() }))
  .mutation(async ({ input }) => {
    return await generatePoliciesFromLLM(openai, input.query);
  });
