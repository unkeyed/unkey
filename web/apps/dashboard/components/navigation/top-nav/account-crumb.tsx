"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { IconUserOutline18 } from "@unkey/icons";
import { CrumbLink } from "./crumb";

export function AccountCrumb() {
  const workspace = useWorkspaceNavigation();

  return (
    <CrumbLink
      icon={<IconUserOutline18 className="size-3.5 text-gray-11" />}
      label="Account settings"
      current
      href={routes.account.overview({ workspaceSlug: workspace.slug })}
    />
  );
}
