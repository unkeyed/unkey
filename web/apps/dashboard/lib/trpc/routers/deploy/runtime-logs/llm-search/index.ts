import { searchClient } from "@/lib/search/client";
import { createSearch } from "@/lib/search/engine";
import { runtimeLogsSearchSpec } from "@/lib/search/specs/runtime-logs";
import { withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

const search = createSearch(runtimeLogsSearchSpec);

export const llmSearch = workspaceProcedure
  .use(withLlmAccess())
  .input(z.object({ query: z.string(), timestamp: z.number() }))
  .mutation(async ({ input, ctx }) => {
    return await search(searchClient(), ctx.validatedQuery, input.timestamp);
  });
