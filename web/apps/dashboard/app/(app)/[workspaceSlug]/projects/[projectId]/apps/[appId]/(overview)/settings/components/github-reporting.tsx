"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import {
  Button,
  SettingsGroup,
  SettingsGroupContent,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useBuildSource } from "../../../../_components/settings/hooks/use-build-source";

export function GitHubReporting() {
  const workspace = useWorkspaceNavigation();
  const { projectId, appId, app, installations, repositoryFullName } = useBuildSource();
  let repositoryLabel = repositoryFullName ?? "Connect a repository in Build settings.";
  if (app?.sourceType === "oci") {
    repositoryLabel = "This app deploys a container image.";
  } else if (installations.isError) {
    repositoryLabel = "Could not load repository details.";
  } else if (!app || installations.isLoading) {
    repositoryLabel = "Loading repository…";
  }

  return (
    <SettingsGroup>
      <SettingsGroupContent>
        <SettingsRow>
          <SettingsRowHeader>
            <SettingsRowTitle>GitHub deployment reporting</SettingsRowTitle>
            <SettingsRowDescription>
              Commit checks and pull request comments use the connected repository, not notification
              rules.
            </SettingsRowDescription>
          </SettingsRowHeader>
          <SettingsRowContent>
            <div className="flex flex-wrap items-center justify-between gap-3">
              <span className="text-sm text-gray-11">{repositoryLabel}</span>
              <Button
                variant="outline"
                size="sm"
                render={
                  <Link
                    href={routes.projects.apps.settings({
                      workspaceSlug: workspace.slug,
                      projectId,
                      appId,
                      page: "build",
                    })}
                  />
                }
              >
                Build settings
              </Button>
            </div>
          </SettingsRowContent>
        </SettingsRow>
      </SettingsGroupContent>
    </SettingsGroup>
  );
}
