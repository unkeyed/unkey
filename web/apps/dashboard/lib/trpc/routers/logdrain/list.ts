import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { decodeLogdrainConfig } from "./config";

const streamSchema = z.enum(["audit_logs"]);

export const listLogdrains = workspaceProcedure.query(async ({ ctx }) => {
  try {
    const rows = await db
      .select({
        id: schema.logdrains.id,
        name: schema.logdrains.name,
        stream: schema.logdrains.stream,
        config: schema.logdrains.config,
        enabled: schema.logdrains.enabled,
        stateStatus: schema.logdrainState.status,
        committedOffsetInsertedAt: schema.logdrainState.committedOffsetInsertedAt,
        consecutiveFailures: schema.logdrainState.consecutiveFailures,
        lastError: schema.logdrainState.lastError,
        createdAt: schema.logdrains.createdAt,
      })
      .from(schema.logdrains)
      .innerJoin(schema.logdrainState, eq(schema.logdrainState.logdrainId, schema.logdrains.id))
      .where(eq(schema.logdrains.workspaceId, ctx.workspace.id));
    const parsed = rows.map(({ config, enabled, stateStatus, ...row }) => {
      const stream = streamSchema.parse(row.stream);
      const destination = decodeLogdrainConfig(config);
      const status: "enabled" | "disabled" | "paused_by_failure" = enabled
        ? stateStatus === "paused_by_failure"
          ? "paused_by_failure"
          : "enabled"
        : "disabled";
      switch (destination.kind) {
        case "http":
          return {
            ...row,
            kind: destination.kind,
            stream,
            status,
            config: {
              url: destination.url,
              format: destination.format,
              headers: destination.headers.map((header) => header.name),
            },
          };
        case "axiom":
          return {
            ...row,
            kind: destination.kind,
            stream,
            status,
            config: {
              dataset: destination.dataset,
              ...(destination.url ? { url: destination.url } : {}),
            },
          };
      }
    });
    return parsed;
  } catch (error) {
    console.error("Failed to list log drains", error);
    throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "Failed to list log drains" });
  }
});
