import { and, db, eq } from "@/lib/db";
import { workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { environments } from "@unkey/db/src/schema";
import { z } from "zod";

export const getEnvironmentBySlug = workspaceProcedure
  .input(
    z.object({
      projectId: z.string().min(1),
      slug: z.string().min(1),
    }),
  )
  .query(async ({ ctx, input }) => {
    const environment = await db.query.environments.findFirst({
      where: and(
        eq(environments.workspaceId, ctx.workspace.id),
        eq(environments.projectId, input.projectId),
        eq(environments.slug, input.slug),
      ),
      columns: { id: true, appId: true },
    });

    if (!environment) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: `Environment '${input.slug}' not found for this project`,
      });
    }

    return { id: environment.id, appId: environment.appId };
  });
