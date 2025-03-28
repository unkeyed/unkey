import { keysQueryListPayload } from "@/app/(app)/apis/[apiId]/keys_v2/[keyAuthId]/_components/components/table/query-logs.schema";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { z } from "zod";
import { getAllKeys } from "./get-all-keys";
import { keyDetailsResponseSchema } from "./schema";

// Define the output schema using existing schemas
const KeysListResponse = z.object({
  keys: z.array(keyDetailsResponseSchema),
  hasMore: z.boolean(),
  nextCursor: z
    .object({
      keyId: z.string(),
    })
    .optional(),
  totalCount: z.number(),
});

type KeysListResponse = z.infer<typeof KeysListResponse>;

const PAGINATION_LIMIT = 50;
export const queryKeysList = rateLimitedProcedure(ratelimit.read)
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
      },
      limit: PAGINATION_LIMIT,
      cursorKeyId: input.cursor?.keyId ?? null,
    });

    const lastKeyId = keys.length > 0 ? keys[keys.length - 1].id : null;

    const response: KeysListResponse = {
      keys: keys,
      hasMore,
      totalCount,
      nextCursor: hasMore && lastKeyId ? { keyId: lastKeyId } : undefined,
    };

    return response;
  });
