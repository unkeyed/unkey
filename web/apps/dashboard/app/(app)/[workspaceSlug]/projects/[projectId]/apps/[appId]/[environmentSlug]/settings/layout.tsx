"use client";

import { routes } from "@/lib/navigation/routes";
import {
  PageContainer,
  PageHeader,
  PageHeaderContent,
  PageHeaderTitle,
  SecondaryNav,
  SecondaryNavGroup,
  SecondaryNavItem,
  SecondaryNavTitle,
} from "@unkey/ui";
import { cn } from "cn";
import Link from "next/link";
import { useSelectedLayoutSegments } from "next/navigation";
import type { ReactNode } from "react";
import { useAppScope } from "../environment-context";
import { EnvironmentSettingsProvider } from "./environment-provider";
import { PreventLeaveProvider } from "./prevent-leave-context";
import { SETTINGS_GROUPS, SETTINGS_SECTIONS } from "./sections";

export default function AppSettingsLayout({ children }: { children: ReactNode }) {
  const scope = useAppScope();
  const segment = useSelectedLayoutSegments().at(0);
  const active =
    SETTINGS_SECTIONS.find((section) => section.page === segment) ?? SETTINGS_SECTIONS[0];

  return (
    <div className="flex min-h-full w-full flex-col md:flex-row">
      <SecondaryNav aria-label="App settings">
        <SecondaryNavTitle>App Settings</SecondaryNavTitle>
        {SETTINGS_GROUPS.map((group) => (
          <SecondaryNavGroup key={group.scope}>
            <div className="hidden px-2 pt-2 pb-1 text-xs font-medium text-gray-10 md:block">
              {group.label}
            </div>
            {SETTINGS_SECTIONS.filter((section) => section.scope === group.scope).map((section) => (
              <SecondaryNavItem
                key={section.label}
                active={section === active}
                className={cn("tone" in section && "text-error-11 hover:text-error-11")}
                render={
                  <Link href={routes.projects.apps.settings({ ...scope, page: section.page })} />
                }
              >
                {section.label}
              </SecondaryNavItem>
            ))}
          </SecondaryNavGroup>
        ))}
      </SecondaryNav>
      <div className="min-w-0 flex-1">
        <PageContainer>
          <PageHeader className="max-w-[920px] px-6">
            <PageHeaderContent>
              <PageHeaderTitle>{active.label}</PageHeaderTitle>
            </PageHeaderContent>
          </PageHeader>
          <PreventLeaveProvider>
            <EnvironmentSettingsProvider>{children}</EnvironmentSettingsProvider>
          </PreventLeaveProvider>
        </PageContainer>
      </div>
    </div>
  );
}
