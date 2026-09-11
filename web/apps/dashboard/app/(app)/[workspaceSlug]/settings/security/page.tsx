import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

export default async function SecuritySettingsPage({
  params,
}: {
  params: Promise<{ workspaceSlug: string }>;
}) {
  const { workspaceSlug } = await params;
  redirect(routes.account.overview({ workspaceSlug }));
}
