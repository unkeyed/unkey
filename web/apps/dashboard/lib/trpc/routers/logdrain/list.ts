import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { db, eq, schema } from "@/lib/db";
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
        status: schema.logdrains.status,
        committedOffsetInsertedAt: schema.logdrains.committedOffsetInsertedAt,
        consecutiveFailures: schema.logdrains.consecutiveFailures,
        createdAt: schema.logdrains.createdAt,
      })
      .from(schema.logdrains)
      .where(eq(schema.logdrains.workspaceId, ctx.workspace.id));
    const parsed = rows.map(({ config, ...row }) => {
      const stream = streamSchema.parse(row.stream);
      const destination = decodeLogdrainConfig(config);
      switch (destination.kind) {
        case "http":
          return {
            ...row,
            kind: destination.kind,
            stream,
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
            config: {
              dataset: destination.dataset,
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
