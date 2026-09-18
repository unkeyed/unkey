import { searchClient } from "@/lib/search/client";
import { createSearch } from "@/lib/search/engine";
import { permissionsSearchSpec } from "@/lib/search/specs/permissions";
import { withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

const search = createSearch(permissionsSearchSpec);

export const permissionsLlmSearch = workspaceProcedure
  .use(withLlmAccess())
  .input(z.object({ query: z.string() }))
  .mutation(async ({ ctx }) => {
    return await search(searchClient(), ctx.validatedQuery);
  });
