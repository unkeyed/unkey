import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../../trpc";

export const deleteProject = workspaceProcedure
  .use(withRatelimit(ratelimit.delete, "trpc_delete"))
  .input(
    z.object({
      projectId: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const project = await db.query.projects.findFirst({
      where: (table, { and, eq }) =>
        and(eq(table.workspaceId, ctx.workspace.id), eq(table.id, input.projectId)),
      columns: {
        id: true,
        name: true,
        deleteProtection: true,
      },
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
        message: "This project has delete protection enabled. Disable it before deleting.",
      });
    }

    try {
      await db.transaction(async (tx) => {
        // Delete dependents first (no FKs in PlanetScale, but keep order defensive)
        await tx
          .delete(schema.ciliumNetworkPolicies)
          .where(eq(schema.ciliumNetworkPolicies.projectId, project.id));
        await tx.delete(schema.vercelBindings).where(eq(schema.vercelBindings.projectId, project.id));
        await tx.delete(schema.instances).where(eq(schema.instances.projectId, project.id));
        await tx.delete(schema.sentinels).where(eq(schema.sentinels.projectId, project.id));
        await tx
          .delete(schema.frontlineRoutes)
          .where(eq(schema.frontlineRoutes.projectId, project.id));
        await tx.delete(schema.customDomains).where(eq(schema.customDomains.projectId, project.id));
        await tx
          .delete(schema.githubRepoConnections)
          .where(eq(schema.githubRepoConnections.projectId, project.id));
        await tx.delete(schema.deployments).where(eq(schema.deployments.projectId, project.id));
        await tx.delete(schema.environments).where(eq(schema.environments.projectId, project.id));

        await tx.delete(schema.projects).where(eq(schema.projects.id, project.id));

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "project.delete",
          description: `Deleted project ${project.id}`,
          resources: [{ type: "project", id: project.id, name: project.name }],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });

      return { projectId: project.id };
    } catch (err) {
      console.error({
        message: "Failed to delete project",
        workspaceId: ctx.workspace.id,
        projectId: input.projectId,
        error: err instanceof Error ? err.message : String(err),
      });
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to delete project. Please try again later or contact support@unkey.com",
      });
    }
  });

