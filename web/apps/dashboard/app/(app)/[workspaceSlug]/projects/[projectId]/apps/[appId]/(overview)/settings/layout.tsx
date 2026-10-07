"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import {
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  SecondaryNav,
  SecondaryNavGroup,
  SecondaryNavItem,
  SecondaryNavTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useSelectedLayoutSegment } from "next/navigation";
import type { ReactNode } from "react";
import { useAppId, useProjectData } from "../data-provider";
import { SETTINGS_SECTIONS, activeSection } from "./sections";

export default function AppSettingsLayout({ children }: { children: ReactNode }) {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useProjectData();
  const appId = useAppId();
  const active = activeSection(useSelectedLayoutSegment());
  const scope = { workspaceSlug: workspace.slug, projectId, appId };

  return (
    <div className="flex min-h-full w-full flex-col md:flex-row">
      <SecondaryNav aria-label="App settings">
        <SecondaryNavTitle>App settings</SecondaryNavTitle>
        <SecondaryNavGroup>
          {SETTINGS_SECTIONS.map((section) => {
            const href = routes.projects.apps.settings({ ...scope, page: section.page });
            return (
              <SecondaryNavItem
                key={section.label}
                active={section === active}
                render={<Link href={href} />}
              >
                {section.label}
              </SecondaryNavItem>
            );
          })}
        </SecondaryNavGroup>
      </SecondaryNav>
      <div className="min-w-0 flex-1">
        <PageContainer>
          <PageHeader className="max-w-[920px] px-6">
            <PageHeaderContent>
              <PageHeaderTitle>{active.label}</PageHeaderTitle>
              {"description" in active ? (
                <PageHeaderDescription>{active.description}</PageHeaderDescription>
              ) : null}
            </PageHeaderContent>
          </PageHeader>
          {children}
        </PageContainer>
      </div>
    </div>
  );
}
