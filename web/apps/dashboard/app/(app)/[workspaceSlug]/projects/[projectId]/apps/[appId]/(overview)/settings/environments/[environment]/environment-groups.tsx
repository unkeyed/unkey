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
import type { ComponentType } from "react";
import { useAppId, useProjectData } from "../../../data-provider";
import { BranchLabel } from "../branch-label";
import { type DomainAssignment, environmentPanelDomains } from "../model";
import { AutoDeploy } from "./auto-deploy";
import { ProductionBranch } from "./production-branch";

const ASSIGNMENT_LABEL: Record<DomainAssignment, string> = {
  live: "Live deployment",
  latest: "Latest deployment",
  "each-branch": "Latest deployment of each branch",
};

const BRANCH_ROW: Record<EnvironmentKind, ComponentType> = {
  production: ProductionBranch,
  preview: PreviewBranch,
};

export function BranchTracking({ environment }: { environment: Environment }) {
  const BranchRow = BRANCH_ROW[environment.kind];
  return (
    <SettingsGroup>
      <SettingsGroupTitle>Branch tracking</SettingsGroupTitle>
      <SettingsGroupContent>
        <BranchRow />
        <AutoDeploy />
      </SettingsGroupContent>
    </SettingsGroup>
  );
}

function PreviewBranch() {
  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>Branch</SettingsRowTitle>
        <SettingsRowDescription>
          Every branch except the production branch deploys here. Each one gets its own URL.
        </SettingsRowDescription>
      </SettingsRowHeader>
      <SettingsRowContent>
        <BranchLabel branch={{ kind: "unassigned" }} className="w-fit" />
      </SettingsRowContent>
    </SettingsRow>
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
