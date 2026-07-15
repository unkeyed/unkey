import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

// The authorization area is a layout + SecondaryNav rail with roles/permissions
// sub-pages, so rendering it faithfully inside a project would need its whole
// layout tree. Redirecting to the workspace-level roles page keeps that UI
// intact; it does swap the sidebar back to workspace scope, an accepted
// trade-off for this projects-first restructure.
export default async function ProjectAuthorizationPage({
  params,
}: {
  params: Promise<{ workspaceSlug: string }>;
}) {
  const { workspaceSlug } = await params;
  redirect(routes.authorization.roles({ workspaceSlug }));
}
