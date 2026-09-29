import { PRODUCTION_ENVIRONMENT_SLUG } from "@/lib/collections/deploy/environments";
import { routes } from "@/lib/navigation/routes";
import { RedirectType, redirect } from "next/navigation";

type Params = { workspaceSlug: string; projectId: string; appId: string };

export default async function AppIndex({ params }: { params: Promise<Params> }) {
  const scope = await params;
  redirect(
    routes.projects.apps.overview({ ...scope, environmentSlug: PRODUCTION_ENVIRONMENT_SLUG }),
    RedirectType.replace,
  );
}
