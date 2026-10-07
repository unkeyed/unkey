"use client";

import { GuardedSlidePanel } from "@/components/guarded-slide-panel";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Environment, EnvironmentKind } from "@/lib/collections/deploy/environments";
import { routes } from "@/lib/navigation/routes";
import { IconArrowUpRightOutline12 } from "@unkey/icons";
import {
  Button,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
  SlidePanelCloseButton,
  SlidePanelContent,
  SlidePanelHeader,
  SlidePanelTitle,
  UnsavedChangesScope,
  useUnsavedChanges,
} from "@unkey/ui";
import Link from "next/link";
import { EnvironmentIcon } from "../../../components/environment-label";
import { useAppId, useProjectData } from "../../data-provider";
import { environmentPanelDomains } from "../../domains/model";
import { AutoDeploy } from "../components/build-settings/auto-deploy-settings";
import { ComputeSettings } from "../components/compute/compute-settings";
import { EnvironmentSettingsScope } from "../environment-provider";
import { useBuildSource } from "../hooks/use-build-source";
import { BranchLabel, branchFor } from "./environments-table";

const BRANCH_NOTE: Record<EnvironmentKind, string> = {
  production:
    "Your repository's default branch. To use a different branch, change the default in your GitHub repository settings.",
  preview: "Every branch except the default deploys here. Each one gets its own URL.",
};

function environmentName(environment: Environment): string {
  return environment.slug.charAt(0).toUpperCase() + environment.slug.slice(1);
}

export function EnvironmentPanel({
  environment,
  isOpen,
  onClose,
  onExitComplete,
}: {
  environment: Environment | null;
  isOpen: boolean;
  onClose: () => void;
  onExitComplete: () => void;
}) {
  const { isDirty, report } = useUnsavedChanges();
  return (
    <GuardedSlidePanel
      dirty={isDirty}
      isOpen={isOpen}
      onClose={onClose}
      onExitComplete={onExitComplete}
    >
      <UnsavedChangesScope report={report}>
        {environment ? (
          <EnvironmentPanelBody key={environment.id} environment={environment} />
        ) : null}
      </UnsavedChangesScope>
    </GuardedSlidePanel>
  );
}

function EnvironmentPanelBody({ environment }: { environment: Environment }) {
  const { hasRepository, defaultBranch } = useBuildSource();
  return (
    <>
      <SlidePanelHeader>
        <SlidePanelTitle className="flex items-center gap-2">
          <EnvironmentIcon kind={environment.kind} className="size-4 text-gray-11" />
          {environmentName(environment)}
        </SlidePanelTitle>
        <SlidePanelCloseButton />
      </SlidePanelHeader>
      <SlidePanelContent className="overflow-y-auto">
        <div className="flex flex-col gap-8 px-6 pt-4 pb-8">
          <EnvironmentSettingsScope environmentId={environment.id} fallback={null}>
            <ComputeSettings />
          </EnvironmentSettingsScope>
          <EnvironmentDomains environment={environment} />
          {hasRepository ? (
            <BranchTracking environment={environment} defaultBranch={defaultBranch} />
          ) : null}
        </div>
      </SlidePanelContent>
    </>
  );
}

function BranchTracking({
  environment,
  defaultBranch,
}: {
  environment: Environment;
  defaultBranch: string | null;
}) {
  return (
    <SettingsGroup>
      <SettingsGroupTitle>Branch tracking</SettingsGroupTitle>
      <SettingsGroupContent>
        <SettingsRow className="flex-row items-center justify-between gap-6">
          <SettingsRowHeader className="min-w-0 shrink">
            <SettingsRowTitle>Branch</SettingsRowTitle>
            <SettingsRowDescription className="max-w-sm text-pretty">
              {BRANCH_NOTE[environment.kind]}
            </SettingsRowDescription>
          </SettingsRowHeader>
          <SettingsRowContent className="flex flex-none items-center justify-end">
            <BranchLabel
              branch={branchFor(environment, true, defaultBranch)}
              className="shrink-0"
            />
          </SettingsRowContent>
        </SettingsRow>
        <EnvironmentSettingsScope environmentId={environment.id} fallback={null}>
          <AutoDeploy environmentName={environmentName(environment)} />
        </EnvironmentSettingsScope>
      </SettingsGroupContent>
    </SettingsGroup>
  );
}

function EnvironmentDomains({ environment }: { environment: Environment }) {
  const { domains, customDomains, projectId } = useProjectData();
  const appId = useAppId();
  const workspace = useWorkspaceNavigation();
  const rows = environmentPanelDomains(environment, domains, customDomains, workspace.slug);
  const domainsHref = routes.projects.apps.domains({
    workspaceSlug: workspace.slug,
    projectId,
    appId,
  });

  return (
    <SettingsGroup>
      <div className="flex items-center justify-between">
        <SettingsGroupTitle>Domains</SettingsGroupTitle>
        <Button variant="outline" size="sm" render={<Link href={domainsHref} />}>
          Manage domains
          <IconArrowUpRightOutline12 className="size-3!" />
        </Button>
      </div>
      <SettingsGroupContent>
        {rows.length === 0 ? (
          <div className="px-5 py-3.5 text-sm text-gray-9">No domains yet.</div>
        ) : (
          rows.map((row) => (
            <div key={row.hostname} className="flex items-center justify-between gap-4 px-5 py-3.5">
              <span className="truncate text-sm font-medium text-gray-12">{row.hostname}</span>
              <span className="shrink-0 text-xs text-gray-11">{row.assignedTo}</span>
            </div>
          ))
        )}
      </SettingsGroupContent>
    </SettingsGroup>
  );
}
