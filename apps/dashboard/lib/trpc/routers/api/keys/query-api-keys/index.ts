import { keysQueryListPayload } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/components/table/query-logs.schema";
import { ratelimit, requireUser, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
import { z } from "zod";
import { getAllKeys } from "./get-all-keys";
import { keyDetailsResponseSchema } from "./schema";

const KeysListResponse = z.object({
  keys: z.array(keyDetailsResponseSchema),
  hasMore: z.boolean(),
  nextCursor: z.string().nullish(),
  totalCount: z.number(),
});

type KeysListResponse = z.infer<typeof KeysListResponse>;

export const queryKeysList = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .use(withRatelimit(ratelimit.read))
  .input(keysQueryListPayload)
  .output(KeysListResponse)
  .query(async ({ ctx, input }) => {
    const { keys, hasMore, totalCount } = await getAllKeys({
      keyspaceId: input.keyAuthId,
      workspaceId: ctx.workspace.id,
      filters: {
        keyIds: input.keyIds,
        names: input.names,
        identities: input.identities,
        tags: input.tags,
      },
      limit: input.limit,
      cursorKeyId: input.cursor ?? null,
    });

    const lastKeyId = keys.length > 0 ? keys[keys.length - 1].id : null;

    const response: KeysListResponse = {
      keys: keys,
      hasMore,
      totalCount,
      nextCursor: hasMore && lastKeyId ? lastKeyId : undefined,
    };

    return response;
  });
