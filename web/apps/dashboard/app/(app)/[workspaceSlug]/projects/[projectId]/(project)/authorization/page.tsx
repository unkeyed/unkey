import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

export default async function ProjectAuthorizationPage({
  params,
}: {
  params: Promise<{ workspaceSlug: string; projectId: string }>;
}) {
  const { workspaceSlug, projectId } = await params;
  redirect(routes.authorization.roles({ workspaceSlug, projectId }));
}
