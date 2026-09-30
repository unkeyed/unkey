import { and, db, desc, eq, isNull } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { apis, keyAuth, projects } from "@unkey/db/src/schema";
import { z } from "zod";

export type ProjectKeyspace = { apiId: string; keyAuthId: string; name: string; keyCount: number };

export const projectKeyspaces = workspaceProcedure
  .input(z.object({ projectId: z.string() }))
  .use(withRatelimit(ratelimit.read))
  .query(async ({ ctx, input }): Promise<ProjectKeyspace[]> => {
    const workspaceId = ctx.workspace.id;

    const project = await db.query.projects.findFirst({
      where: and(eq(projects.workspaceId, workspaceId), eq(projects.id, input.projectId)),
      columns: { id: true },
    });
    if (!project) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Project not found" });
    }

    return db
      .select({
        apiId: apis.id,
        keyAuthId: keyAuth.id,
        name: apis.name,
        keyCount: keyAuth.sizeApprox,
      })
      .from(apis)
      .innerJoin(keyAuth, eq(keyAuth.id, apis.keyAuthId))
      .where(
        and(
          eq(apis.workspaceId, workspaceId),
          eq(apis.projectId, project.id),
          isNull(apis.deletedAtM),
        ),
      )
      .orderBy(desc(keyAuth.sizeApprox));
  });
