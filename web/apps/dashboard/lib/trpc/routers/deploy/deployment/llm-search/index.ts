import { searchClient } from "@/lib/search/client";
import { createSearch } from "@/lib/search/engine";
import { deploymentsSearchSpec } from "@/lib/search/specs/deployments";
import { withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

const search = createSearch(deploymentsSearchSpec);

export const searchDeployments = workspaceProcedure
  .use(withLlmAccess())
  .input(z.object({ query: z.string() }))
  .mutation(async ({ ctx }) => {
    return await search(searchClient(), ctx.validatedQuery);
  });
