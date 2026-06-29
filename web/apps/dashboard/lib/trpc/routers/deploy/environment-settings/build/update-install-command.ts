import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appBuildSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export const updateInstallCommand = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      // Empty means "let Railpack auto-detect the install command".
      installCommand: z.string().trim().max(1000),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const env = await db.query.environments.findFirst({
      where: and(
        eq(environments.id, input.environmentId),
        eq(environments.workspaceId, ctx.workspace.id),
      ),
      columns: { appId: true },
    });
    if (!env) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
    }

    // NULL is the canonical "auto-detect" representation.
    const installCommand = input.installCommand === "" ? null : input.installCommand;

    await db
      .insert(appBuildSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        installCommand,
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { installCommand, updatedAt: Date.now() } });
  });
