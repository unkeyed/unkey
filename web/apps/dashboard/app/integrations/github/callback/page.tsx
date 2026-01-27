import { getAuth } from "@/lib/auth";
import { db, eq, schema } from "@/lib/db";
import { githubAppEnv } from "@/lib/env";
import { newId } from "@unkey/id";
import { redirect } from "next/navigation";

type Props = {
  searchParams: {
    installation_id?: string;
    setup_action?: string;
    state?: string;
  };
};

export default async function Page(props: Props) {
  const env = githubAppEnv();
  if (!env) {
    return <div>GitHub App not configured</div>;
  }

  const { installation_id, state } = props.searchParams;

  if (!installation_id) {
    return <div>Missing installation_id</div>;
  }

  if (!state) {
    return <div>Missing state parameter</div>;
  }

  const [projectId, workspaceSlug] = state.split(":");
  if (!projectId || !workspaceSlug) {
    return <div>Invalid state parameter</div>;
  }

  const { orgId } = await getAuth();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return <div>Workspace not found</div>;
  }

  const project = await db.query.projects.findFirst({
    where: (table, { and, eq }) =>
      and(eq(table.id, projectId), eq(table.workspaceId, workspace.id)),
  });

  if (!project) {
    return <div>Project not found</div>;
  }

  const existingInstallation = await db.query.githubAppInstallations.findFirst({
    where: eq(schema.githubAppInstallations.projectId, projectId),
  });

  const installationIdNum = Number.parseInt(installation_id, 10);
  if (Number.isNaN(installationIdNum)) {
    return <div>Invalid installation_id</div>;
  }

  if (existingInstallation) {
    await db
      .update(schema.githubAppInstallations)
      .set({ installationId: installationIdNum })
      .where(eq(schema.githubAppInstallations.projectId, projectId));
  } else {
    await db.insert(schema.githubAppInstallations).values({
      id: newId("github"),
      projectId,
      installationId: installationIdNum,
      repositoryId: 0,
      repositoryFullName: "",
      createdAt: Date.now(),
      updatedAt: null,
    });
  }

  redirect(`/${workspaceSlug}/projects/${projectId}/github?installed=true`);
}
