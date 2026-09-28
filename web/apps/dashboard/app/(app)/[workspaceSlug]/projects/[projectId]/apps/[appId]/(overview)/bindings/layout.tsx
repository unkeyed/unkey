import { privateNetworking } from "@/lib/flags";
import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";
import type { ReactNode } from "react";

export default async function BindingsLayout({
  children,
  params,
}: {
  children: ReactNode;
  params: Promise<{ workspaceSlug: string; projectId: string; appId: string }>;
}) {
  if (!(await privateNetworking())) {
    const { workspaceSlug, projectId, appId } = await params;
    redirect(routes.projects.apps.overview({ workspaceSlug, projectId, appId }));
  }
  return children;
}
