import { routes } from "@/lib/navigation/routes";
import { SettingsGroups } from "@unkey/ui";
import { redirect } from "next/navigation";
import { EnvironmentsTable } from "./environments-table";

export default async function EnvironmentsSettingsPage({
  params,
  searchParams,
}: {
  params: Promise<{ workspaceSlug: string; projectId: string; appId: string }>;
  searchParams: Promise<{ environment?: string | string[] }>;
}) {
  const scope = await params;
  const { environment } = await searchParams;
  if (typeof environment === "string") {
    redirect(routes.projects.apps.settings({ ...scope, page: "environments", environment }));
  }

  return (
    <SettingsGroups>
      <EnvironmentsTable />
    </SettingsGroups>
  );
}
