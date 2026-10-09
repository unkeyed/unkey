"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Environment, EnvironmentKind } from "@/lib/collections/deploy/environments";
import { routes } from "@/lib/navigation/routes";
import { IconArrowUpRightOutline12 } from "@unkey/icons";
import {
  Button,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemTitle,
  SettingsGroup,
  SettingsGroupContent,
  SettingsGroupTitle,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useAppId, useProjectData } from "../../../data-provider";
import { AutoDeploy } from "../../components/build-settings/auto-deploy-settings";
import { environmentName } from "../../sections";
import type { Branch } from "../branch";
import { BranchLabel } from "../branch-label";
import { type DomainAssignment, environmentPanelDomains } from "../model";

const ASSIGNMENT_LABEL: Record<DomainAssignment, string> = {
  live: "Live deployment",
  latest: "Latest deployment",
  "each-branch": "Latest deployment of each branch",
};

const BRANCH_NOTE: Record<EnvironmentKind, string> = {
  production:
    "Your repository's default branch. To use a different branch, change the default in your GitHub repository settings.",
  preview: "Every branch except the default deploys here. Each one gets its own URL.",
};

export function BranchTracking({
  environment,
  branch,
}: { environment: Environment; branch: Branch }) {
  return (
    <SettingsGroup>
      <SettingsGroupTitle>Branch tracking</SettingsGroupTitle>
      <SettingsGroupContent>
        <SettingsRow>
          <SettingsRowHeader>
            <SettingsRowTitle>Branch</SettingsRowTitle>
            <SettingsRowDescription>{BRANCH_NOTE[environment.kind]}</SettingsRowDescription>
          </SettingsRowHeader>
          <SettingsRowContent>
            <BranchLabel branch={branch} className="w-fit" />
          </SettingsRowContent>
        </SettingsRow>
        <AutoDeploy environmentName={environmentName(environment.slug)} />
      </SettingsGroupContent>
    </SettingsGroup>
  );
}

export function EnvironmentDomains({ environment }: { environment: Environment }) {
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
          <Item>
            <ItemContent>
              <ItemDescription>No domains yet.</ItemDescription>
            </ItemContent>
          </Item>
        ) : (
          rows.map((row) => (
            <Item key={row.hostname}>
              <ItemContent>
                <ItemTitle className="truncate">{row.hostname}</ItemTitle>
              </ItemContent>
              <ItemActions className="text-xs text-gray-11">
                {ASSIGNMENT_LABEL[row.assignment]}
              </ItemActions>
            </Item>
          ))
        )}
      </SettingsGroupContent>
    </SettingsGroup>
  );
}
