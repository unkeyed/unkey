import { searchClient } from "@/lib/search/client";
import { createSearch } from "@/lib/search/engine";
import { rolesSearchSpec } from "@/lib/search/specs/roles";
import { withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

const search = createSearch(rolesSearchSpec);

export const rolesLlmSearch = workspaceProcedure
  .use(withLlmAccess())
  .input(z.object({ query: z.string() }))
  .mutation(async ({ ctx }) => {
    return await search(searchClient(), ctx.validatedQuery);
  });
