import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appBuildSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

const dockerContextSegment = /^[A-Za-z0-9._-]+$/;

const dockerContextSchema = z
  .string()
  .min(1)
  .refine(
    (path) =>
      path === "." ||
      (path === path.trim() &&
        !path.startsWith("/") &&
        !path.includes("\\") &&
        path
          .split("/")
          .every(
            (segment) => segment !== "." && segment !== ".." && dockerContextSegment.test(segment),
          )),
    "Docker context must be a repository-relative path like 'api' or 'services/api'.",
  );

export const updateDockerContext = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      dockerContext: dockerContextSchema,
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

    await db
      .insert(appBuildSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        dockerContext: input.dockerContext,
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { dockerContext: input.dockerContext, updatedAt: Date.now() } });
  });
