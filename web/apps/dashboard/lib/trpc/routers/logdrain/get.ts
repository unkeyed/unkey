import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { and, db, eq, schema } from "@/lib/db";
import { workspaceProcedure } from "../../trpc";
import { decodeLogdrainConfig } from "./config";

export const getLogdrain = workspaceProcedure
  .input(z.object({ id: z.string().min(1) }))
  .query(async ({ ctx, input }) => {
    try {
      const [row] = await db
        .select({
          id: schema.logdrains.id,
          name: schema.logdrains.name,
          config: schema.logdrains.config,
          status: schema.logdrains.status,
        })
        .from(schema.logdrains)
        .where(
          and(
            eq(schema.logdrains.id, input.id),
            eq(schema.logdrains.workspaceId, ctx.workspace.id),
          ),
        );
      if (!row) {
        throw new TRPCError({ code: "NOT_FOUND", message: "Log drain not found" });
      }

      const destination = decodeLogdrainConfig(row.config);
      switch (destination.kind) {
        case "http":
          return {
            id: row.id,
            name: row.name,
            kind: destination.kind,
            status: row.status,
            config: {
              url: destination.url,
              format: destination.format,
              headers: destination.headers.map((header) => header.name),
            },
          };
        case "axiom":
          return {
            id: row.id,
            name: row.name,
            kind: destination.kind,
            status: row.status,
            config: {
              dataset: destination.dataset,
            },
          };
      }
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to get log drain", error);
      throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "Failed to get log drain" });
    }
  });
