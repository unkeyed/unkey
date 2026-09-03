import { getAuth } from "@/lib/auth";
import { and, db, eq, isNull, schema } from "@/lib/db";
import { routes } from "@/lib/navigation/routes";
import { notFound, redirect } from "next/navigation";

export const dynamic = "force-dynamic";

export default async function AlertRedirectPage({
  params,
}: {
  params: Promise<{ workspaceSlug: string; alertId: string }>;
}) {
  const { workspaceSlug, alertId } = await params;
  const { orgId } = await getAuth();
  if (!orgId) {
    notFound();
  }

  const [alert] = await db
    .select({
      projectId: schema.alertEvents.projectId,
      appId: schema.alertEvents.appId,
      environmentId: schema.alertEvents.environmentId,
    })
    .from(schema.alertEvents)
    .innerJoin(
      schema.workspaces,
      and(
        eq(schema.workspaces.id, schema.alertEvents.workspaceId),
        eq(schema.workspaces.orgId, orgId),
        eq(schema.workspaces.slug, workspaceSlug),
        isNull(schema.workspaces.deletedAtM),
      ),
    )
    .where(eq(schema.alertEvents.id, alertId))
    .limit(1);

  if (!alert) {
    notFound();
  }

  redirect(
    routes.projects.apps.anomalies({
      workspaceSlug,
      projectId: alert.projectId,
      appId: alert.appId,
      environmentId: alert.environmentId,
      alertId,
    }),
  );
}
