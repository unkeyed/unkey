import { getAuth } from "@/lib/auth";
import { db, schema } from "@/lib/db";
import { githubAppEnv } from "@/lib/env";
import { type NextRequest, NextResponse } from "next/server";

export async function GET(request: NextRequest) {
  const env = githubAppEnv();
  if (!env) {
    return NextResponse.json({ error: "GitHub App not configured" }, { status: 500 });
  }

  const searchParams = request.nextUrl.searchParams;
  const installation_id = searchParams.get("installation_id");
  const state = searchParams.get("state");

  if (!installation_id) {
    return NextResponse.json({ error: "Missing installation_id" }, { status: 400 });
  }

  if (!state) {
    return NextResponse.json({ error: "Missing state parameter" }, { status: 400 });
  }

  let parsedState: { projectId?: string };
  try {
    parsedState = JSON.parse(state);
  } catch {
    return NextResponse.json({ error: "Invalid state parameter" }, { status: 400 });
  }

  const { projectId } = parsedState;
  if (!projectId) {
    return NextResponse.json({ error: "Missing projectId in state" }, { status: 400 });
  }

  const { orgId } = await getAuth();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return NextResponse.json({ error: "Workspace not found" }, { status: 404 });
  }

  const project = await db.query.projects.findFirst({
    where: (table, { and, eq }) =>
      and(eq(table.id, projectId), eq(table.workspaceId, workspace.id)),
  });

  if (!project) {
    return NextResponse.json({ error: "Project not found" }, { status: 404 });
  }

  const installationIdNum = Number.parseInt(installation_id, 10);
  if (Number.isNaN(installationIdNum)) {
    return NextResponse.json({ error: "Invalid installation_id" }, { status: 400 });
  }

  const existingInstallation = await db.query.githubAppInstallations.findFirst({
    where: (table, { and, eq }) =>
      and(eq(table.workspaceId, workspace.id), eq(table.installationId, installationIdNum)),
  });

  if (!existingInstallation) {
    await db.insert(schema.githubAppInstallations).values({
      workspaceId: workspace.id,
      installationId: installationIdNum,
      createdAt: Date.now(),
      updatedAt: null,
    });
  }

  return NextResponse.redirect(
    new URL(`/${workspace.slug}/projects/${projectId}/settings?installed=true`, request.url),
  );
}
