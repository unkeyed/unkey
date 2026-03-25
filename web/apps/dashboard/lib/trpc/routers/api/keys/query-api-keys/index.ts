import { apiKeysQueryPayload as keysQueryListPayload } from "@/components/api-keys-table/schema/api-keys.schema";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { z } from "zod";
import { getAllKeys } from "./get-all-keys";
import { keyDetailsResponseSchema } from "./schema";

const KeysListResponse = z.object({
  keys: z.array(keyDetailsResponseSchema),
  totalCount: z.number(),
});

export const queryKeysList = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(keysQueryListPayload)
  .output(KeysListResponse)
  .query(async ({ ctx, input }) => {
    const { keys, totalCount } = await getAllKeys({
      keyspaceId: input.keyAuthId,
      workspaceId: ctx.workspace.id,
      filters: {
        keyIds: input.keyIds,
        names: input.names,
        identities: input.identities,
        tags: input.tags,
      },
      limit: input.limit,
      page: input.page,
    });

    return { keys, totalCount };
  });
