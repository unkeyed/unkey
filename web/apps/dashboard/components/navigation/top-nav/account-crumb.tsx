"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { User } from "@unkey/icons";
import { CrumbLink } from "./crumb";

export function AccountCrumb() {
  const workspace = useWorkspaceNavigation();

  return (
    <CrumbLink
      icon={<User className="size-3.5 text-accent-11" iconSize="sm-regular" />}
      label="Account settings"
      current
      href={routes.account.overview({ workspaceSlug: workspace.slug })}
    />
  );
}
