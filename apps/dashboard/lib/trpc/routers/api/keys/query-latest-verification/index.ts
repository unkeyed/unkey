import { clickhouse } from "@/lib/clickhouse";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const lastVerificationTimePayload = z.object({
  keyAuthId: z.string(),
  keyId: z.string(),
});

// tRPC endpoint for getting just the time of the latest verification
export const keyLastVerificationTime = workspaceProcedure
  .input(lastVerificationTimePayload)
  .query(async ({ ctx, input }) => {
    const result = await clickhouse.verifications.latest({
      workspaceId: ctx.workspace.id,
      keySpaceId: input.keyAuthId,
      keyId: input.keyId,
      limit: 1,
    });

    if (!result || result.err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from ClickHouse.",
      });
    }

    return {
      lastVerificationTime: result.val.length > 0 ? result.val[0].time : null,
    };
  });
