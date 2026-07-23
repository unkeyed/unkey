import { and, db, eq, schema } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const updateDockerImage = workspaceProcedure
  .input(
    z.object({
      appId: z.string().min(1),
      image: z.string().trim().min(1, "Image is required").max(512, "Image is too long"),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const app = await db.query.apps.findFirst({
      where: (table, { and: andFn, eq: eqFn }) =>
        andFn(eqFn(table.id, input.appId), eqFn(table.workspaceId, ctx.workspace.id)),
      columns: { id: true, sourceType: true },
    });

    if (!app) {
      throw new TRPCError({ code: "NOT_FOUND", message: "App not found" });
    }
    if (app.sourceType !== "docker_image") {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Docker images can only be configured for Docker image apps",
      });
    }

    const now = Date.now();
    await db.transaction(async (tx) => {
      await tx
        .insert(schema.appDockerSources)
        .values({
          workspaceId: ctx.workspace.id,
          appId: input.appId,
          image: input.image,
          createdAt: now,
          updatedAt: null,
        })
        .onDuplicateKeyUpdate({ set: { image: input.image, updatedAt: now } });
      await tx
        .update(schema.apps)
        .set({ updatedAt: now })
        .where(and(eq(schema.apps.id, input.appId), eq(schema.apps.workspaceId, ctx.workspace.id)));
    });

    return { success: true };
  });
