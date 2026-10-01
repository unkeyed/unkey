import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

export default async function BindingsRedirect({
  params,
}: {
  params: Promise<{ workspaceSlug: string; projectId: string; appId: string }>;
}) {
  redirect(routes.projects.apps.connections(await params));
}
