import { insertAuditLogs } from "@/lib/audit";
import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { getCtrlClients } from "../../ctrl";

// resourceType lists every resource that supports scheduled deletion.
// Adding a new resource means: extend this enum, add a case in the
// switch below, and add a "<resource>.restore" entry to
// web/internal/schema/src/auditlog.ts.
const resourceType = z.enum(["project"]);

type Ctx = Parameters<Parameters<typeof workspaceProcedure.mutation>[0]>[0]["ctx"];

export const restoreResource = workspaceProcedure
  .input(
    z.object({
      resourceType,
      resourceId: z.string(),
    }),
  )
  .use(withRatelimit(ratelimit.delete))
  .mutation(async ({ ctx, input }) => {
    switch (input.resourceType) {
      case "project":
        return await restoreProject(ctx, input.resourceId);
      default:
        // exhaustiveness: this branch is unreachable as long as every
        // resourceType variant has a case above. The cast surfaces the
        // missing branch as a TS error if a new variant is added.
        return input.resourceType satisfies never;
    }
  });

async function restoreProject(ctx: Ctx, projectId: string) {
  const project = await db.query.projects.findFirst({
    where: (table, { and, eq, isNotNull }) =>
      and(
        eq(table.id, projectId),
        eq(table.workspaceId, ctx.workspace.id),
        isNotNull(table.deletionId),
      ),
  });

  if (!project) {
    // Either the project never existed, isn't in this workspace, or
    // was already restored / hard-deleted. Same error either way; we
    // don't leak which one.
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "No scheduled deletion found for that project",
    });
  }

  const ctrl = getCtrlClients();
  try {
    await ctrl.project.restoreProject({ projectId });
  } catch (err) {
    if (err instanceof TRPCError) {
      throw err;
    }
    console.error("Failed to restore project:", err);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to restore project",
    });
  }

  await insertAuditLogs(db, {
    workspaceId: ctx.workspace.id,
    actor: { type: "user", id: ctx.user.id },
    event: "project.restore",
    description: `Restored ${project.id}`,
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

  return { success: true, resourceId: project.id };
}
