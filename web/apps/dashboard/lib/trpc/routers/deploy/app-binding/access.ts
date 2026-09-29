import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";

export async function requireProject(workspaceId: string, projectId: string) {
  const row = await db.query.projects.findFirst({
    columns: { id: true },
    where: (t, { and, eq }) => and(eq(t.id, projectId), eq(t.workspaceId, workspaceId)),
  });
  if (!row) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Project not found." });
  }
}
export async function requireApp(workspaceId: string, projectId: string, appId: string) {
  const row = await db.query.apps.findFirst({
    columns: { id: true, slug: true },
    where: (t, { and, eq }) =>
      and(eq(t.id, appId), eq(t.projectId, projectId), eq(t.workspaceId, workspaceId)),
  });
  if (!row) {
    throw new TRPCError({ code: "NOT_FOUND", message: "App not found." });
  }
  return row;
}
export async function requireBinding(workspaceId: string, projectId: string, id: string) {
  const row = await db.query.appBindings.findFirst({
    columns: {
      id: true,
      appId: true,
      environmentId: true,
      resourceId: true,
      name: true,
    },
    where: (t, { and, eq }) =>
      and(
        eq(t.id, id),
        eq(t.workspaceId, workspaceId),
        eq(t.projectId, projectId),
        eq(t.resourceType, "app"),
      ),
  });
  if (!row) {
    throw new TRPCError({ code: "NOT_FOUND", message: "Binding not found." });
  }
  return row;
}
