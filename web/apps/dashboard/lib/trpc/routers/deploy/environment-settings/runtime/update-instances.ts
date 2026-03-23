import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRegionalSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateInstances = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      replicasPerRegion: z
        .number()
        .min(1)
        .max(
          4,
          "Instances are limited to 4 per region during beta. Please contact support@unkey.com if you need more.",
        ),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    await db
      .update(appRegionalSettings)
      .set({
        replicas: input.replicasPerRegion,
      })
      .where(
        and(
          eq(appRegionalSettings.workspaceId, ctx.workspace.id),
          eq(appRegionalSettings.environmentId, input.environmentId),
        ),
      )
      .catch((err) => {
        console.error(err);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "Unable to update instances.",
        });
      });
  });
