import { routes } from "@/lib/navigation/routes";
import { permanentRedirect } from "next/navigation";

export default async function RuntimeSettingsPage({
  params,
}: {
  params: Promise<{ workspaceSlug: string; projectId: string; appId: string }>;
}) {
  const { workspaceSlug, projectId, appId } = await params;
  permanentRedirect(
    routes.projects.apps.settings({ workspaceSlug, projectId, appId, page: "environments" }),
  );
}
