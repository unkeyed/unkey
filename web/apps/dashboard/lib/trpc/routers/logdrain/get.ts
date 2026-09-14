import { and, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
import { decodeLogdrainConfig, toPublicLogdrainConfig } from "./config";

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
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Log drain not found",
        });
      }

      return {
        id: row.id,
        name: row.name,
        status: row.status,
        ...toPublicLogdrainConfig(decodeLogdrainConfig(row.config)),
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to get log drain", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to get log drain",
      });
    }
  });
