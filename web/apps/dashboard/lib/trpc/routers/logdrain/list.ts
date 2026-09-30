import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { decodeLogdrainConfig, toPublicLogdrainConfig } from "./config";

const streamSchema = z.enum([
  "audit_logs",
  "key_verifications",
  "gateway_requests",
  "runtime_logs",
  "ratelimits",
]);

export const listLogdrains = workspaceProcedure.query(async ({ ctx }) => {
  try {
    const rows = await db
      .select({
        id: schema.logdrains.id,
        name: schema.logdrains.name,
        stream: schema.logdrains.stream,
        config: schema.logdrains.config,
        status: schema.logdrains.status,
        committedOffsetInsertedAt: schema.logdrains.committedOffsetInsertedAt,
        consecutiveFailures: schema.logdrains.consecutiveFailures,
        createdAt: schema.logdrains.createdAt,
      })
      .from(schema.logdrains)
      .where(eq(schema.logdrains.workspaceId, ctx.workspace.id));
    return rows.map(({ config, ...row }) => ({
      ...row,
      ...toPublicLogdrainConfig(decodeLogdrainConfig(config)),
      stream: streamSchema.parse(row.stream),
    }));
  } catch (error) {
    console.error("Failed to list log drains", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to list log drains",
    });
  }
});
