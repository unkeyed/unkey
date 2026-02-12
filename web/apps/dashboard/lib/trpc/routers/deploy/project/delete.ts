import { insertAuditLogs } from "@/lib/audit";
import { db, eq, inArray, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const deleteProject = workspaceProcedure
  .input(
    z.object({
      projectId: z.string(),
    }),
  )
  .use(withRatelimit(ratelimit.delete))
  .mutation(async ({ ctx, input }) => {
    const workspace = ctx.workspace;

    const project = await db.query.projects.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.id, input.projectId), eq(table.workspaceId, workspace.id)),
    });

    if (!project) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "Project not found",
      });
    }

    if (project.deleteProtection) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "Cannot delete project with delete protection enabled",
      });
    }

    await db.transaction(async (tx) => {
      // 1. Delete environment variables (via environments)
      const environments = await tx.query.environments.findMany({
        where: (table, { eq }) => eq(table.projectId, input.projectId),
        columns: { id: true },
      });

      if (environments.length > 0) {
        const environmentIds = environments.map((e) => e.id);
        await tx
          .delete(schema.environmentVariables)
          .where(inArray(schema.environmentVariables.environmentId, environmentIds));
      }

      // Delete environments
      await tx
        .delete(schema.environments)
        .where(eq(schema.environments.projectId, input.projectId));

      // Delete instances (via deployments)
      const deployments = await tx.query.deployments.findMany({
        where: (table, { eq }) => eq(table.projectId, input.projectId),
        columns: { id: true },
      });

      if (deployments.length > 0) {
        const deploymentIds = deployments.map((d) => d.id);
        await tx
          .delete(schema.instances)
          .where(inArray(schema.instances.deploymentId, deploymentIds));
      }

      // Delete sentinels (by projectId)
      await tx.delete(schema.sentinels).where(eq(schema.sentinels.projectId, input.projectId));

      // Delete deployments
      await tx.delete(schema.deployments).where(eq(schema.deployments.projectId, input.projectId));

      // Delete custom domains, frontline routes, github connections
      await tx
        .delete(schema.customDomains)
        .where(eq(schema.customDomains.projectId, input.projectId));
      await tx
        .delete(schema.frontlineRoutes)
        .where(eq(schema.frontlineRoutes.projectId, input.projectId));
      await tx
        .delete(schema.githubRepoConnections)
        .where(eq(schema.githubRepoConnections.projectId, input.projectId));

      await tx.delete(schema.projects).where(eq(schema.projects.id, input.projectId));

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspace.id,
        actor: { type: "user", id: ctx.user.id },
        event: "project.delete",
        description: `Deleted ${project.id}`,
        resources: [
          {
            type: "project",
            id: project.id,
            name: project.name,
          },
        ],
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });

    return { success: true, projectId: input.projectId };
  });
