import { clickhouse } from "@/lib/clickhouse";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const lastVerificationTimePayload = z.object({
  identityId: z.string(),
});

export const identityLastVerificationTime = workspaceProcedure
  .input(lastVerificationTimePayload)
  .query(async ({ ctx, input }) => {
    try {
      const query = clickhouse.querier.query({
        query: `
          SELECT max(toUnixTimestamp(time) * 1000) as last_used
          FROM default.key_verifications_per_minute_v3
          WHERE workspace_id = {workspaceId: String}
            AND identity_id = {identityId: String}
        `,
        params: z.object({
          workspaceId: z.string(),
          identityId: z.string(),
        }),
        schema: z.object({
          last_used: z.number().nullable(),
        }),
      });

      const result = await query({
        workspaceId: ctx.workspace.id,
        identityId: input.identityId,
      });

      if (result.err) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Something went wrong when fetching data from ClickHouse.",
        });
      }

      return {
        lastVerificationTime: result.val[0]?.last_used ?? null,
      };
    } catch (error) {
      console.error("Error querying last verification for identity:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Something went wrong when fetching data from ClickHouse.",
      });
    }
  });
