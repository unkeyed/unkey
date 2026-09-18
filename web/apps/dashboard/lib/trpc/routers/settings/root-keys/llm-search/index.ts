import { searchClient } from "@/lib/search/client";
import { createSearch } from "@/lib/search/engine";
import { rootKeysSearchSpec } from "@/lib/search/specs/root-keys";
import { requireWorkspaceAdmin, withLlmAccess, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";

const search = createSearch(rootKeysSearchSpec);

export const rootKeysLlmSearch = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .use(withLlmAccess())
  .input(z.object({ query: z.string() }))
  .mutation(async ({ input }) => {
    return await search(searchClient(), input.query);
  });
