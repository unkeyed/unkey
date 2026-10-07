"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Environment } from "@/lib/collections/deploy/environments";
import { routes } from "@/lib/navigation/routes";
import { SettingsGroups } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";
import { useAppId, useProjectData } from "../../data-provider";
import { EnvironmentPanel } from "./environment-panel";
import { EnvironmentsTable } from "./environments-table";

export default function EnvironmentsSettingsPage() {
  const { environments, projectId } = useProjectData();
  const appId = useAppId();
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const slug = useSearchParams().get("environment");
  const selected = environments.find((e) => e.slug === slug) ?? null;
  const [shown, setShown] = useState<Environment | null>(selected);
  if (selected && selected.id !== shown?.id) {
    setShown(selected);
  }

  const close = () =>
    router.replace(
      routes.projects.apps.settings({
        workspaceSlug: workspace.slug,
        projectId,
        appId,
        page: "environments",
      }),
      { scroll: false },
    );

  return (
    <SettingsGroups>
      <EnvironmentsTable />
      <EnvironmentPanel
        environment={selected ?? shown}
        isOpen={selected !== null}
        onClose={close}
        onExitComplete={() => setShown(null)}
      />
    </SettingsGroups>
  );
}
