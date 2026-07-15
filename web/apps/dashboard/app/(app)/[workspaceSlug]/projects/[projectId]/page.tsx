import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

// Overview is the project landing. Redirecting the bare project URL catches
// every "land on the project" entry point (cards, crumbs, stale links) in one
// place; the Apps list lives at its own /apps route.
export default async function ProjectPage({
  params,
}: {
  params: Promise<{ workspaceSlug: string; projectId: string }>;
}) {
  const { workspaceSlug, projectId } = await params;
  redirect(routes.projects.overview({ workspaceSlug, projectId }));
}
