"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import {
  PageContainer,
  PageHeader,
  PageHeaderBack,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
  SecondaryNav,
  SecondaryNavGroup,
  SecondaryNavItem,
  SecondaryNavTitle,
} from "@unkey/ui";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import type { ReactNode } from "react";
import { useAppId, useProjectData } from "../data-provider";
import {
  SETTINGS_RAIL,
  SETTINGS_SECTIONS,
  activeSection,
  sectionPage,
  settingsHeader,
} from "./sections";

export default function AppSettingsLayout({ children }: { children: ReactNode }) {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useProjectData();
  const appId = useAppId();
  const active = activeSection(useSelectedLayoutSegments());
  const { back, title, description } = settingsHeader(active);
  const scope = { workspaceSlug: workspace.slug, projectId, appId };

  return (
    <div className="flex min-h-full w-full flex-col md:flex-row">
      <SecondaryNav aria-label="App settings">
        <SecondaryNavTitle>App settings</SecondaryNavTitle>
        <SecondaryNavGroup>
          {SETTINGS_RAIL.map((section) => {
            const href = routes.projects.apps.settings({ ...scope, page: sectionPage(section) });
            return (
              <SecondaryNavItem
                key={section}
                active={section === active}
                render={<Link href={href} />}
              >
                {SETTINGS_SECTIONS[section].label}
              </SecondaryNavItem>
            );
          })}
        </SecondaryNavGroup>
      </SecondaryNav>
      <div className="min-w-0 flex-1">
        <PageContainer>
          <PageHeader className="max-w-[920px] px-6">
            <PageHeaderContent>
              {back ? (
                <PageHeaderBack
                  render={<Link href={routes.projects.apps.settings({ ...scope, page: back })} />}
                >
                  {SETTINGS_SECTIONS[back].label}
                </PageHeaderBack>
              ) : null}
              <PageHeaderTitle>{title}</PageHeaderTitle>
              {description ? <PageHeaderDescription>{description}</PageHeaderDescription> : null}
            </PageHeaderContent>
          </PageHeader>
          {children}
        </PageContainer>
      </div>
    </div>
  );
}
