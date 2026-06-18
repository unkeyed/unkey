"use client";
import { Navigation } from "@/components/navigation/navigation";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { Layers3 } from "@unkey/icons";
import { LogsClient } from "./components/logs-client";

export default function Page() {
  const workspace = useWorkspaceNavigation();
  return (
    <div>
      <Navigation
        href={routes.logs.list({ workspaceSlug: workspace.slug })}
        name="Logs"
        icon={<Layers3 />}
      />
      <LogsClient />
    </div>
  );
}
