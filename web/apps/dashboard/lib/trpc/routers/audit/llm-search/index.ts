import { db } from "@/lib/db";
import { env } from "@/lib/env";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import OpenAI from "openai";
import { z } from "zod";
import { getStructuredAuditSearchFromLLM } from "./utils";

const openai = env().OPENAI_API_KEY
  ? new OpenAI({
      apiKey: env().OPENAI_API_KEY,
    })
  : null;

export const auditLogsSearch = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ query: z.string(), timestamp: z.number() }))
  .mutation(async ({ ctx, input }) => {
    const workspace = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.orgId, ctx.tenant.id), isNull(table.deletedAtM)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to verify workspace access. Please try again or contact support@unkey.com if this persists.",
        });
      });

    if (!workspace) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Workspace not found, please contact support using support@unkey.com.",
      });
    }

    return await getStructuredAuditSearchFromLLM(openai, input.query, input.timestamp);
  });
