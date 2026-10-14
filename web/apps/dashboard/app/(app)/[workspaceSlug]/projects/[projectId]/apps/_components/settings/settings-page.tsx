"use client";

import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import {
  SettingsGroup,
  SettingsGroupCollapsible,
  SettingsGroupContent,
  SettingsGroupTitle,
} from "@unkey/ui";
import { Fragment, type ReactNode } from "react";
import { useEnvironmentSettings } from "./environment-provider";

export type SettingsPage = {
  title: string;
  description: string;
  summary: (settings: EnvironmentSettings) => string;
  groups: ReadonlyArray<{ title: string; rows: ReactNode }>;
};

export function SettingsPageGroups({ page }: { page: SettingsPage }) {
  return page.groups.map((group) => (
    <SettingsGroup key={group.title}>
      <SettingsGroupTitle>{group.title}</SettingsGroupTitle>
      <SettingsGroupContent>{group.rows}</SettingsGroupContent>
    </SettingsGroup>
  ));
}

export function SettingsPageCollapsible({ page }: { page: SettingsPage }) {
  const { settings } = useEnvironmentSettings();
  return (
    <SettingsGroupCollapsible
      title={page.title}
      description={page.description}
      summary={page.summary(settings)}
    >
      {page.groups.map((group) => (
        <Fragment key={group.title}>{group.rows}</Fragment>
      ))}
    </SettingsGroupCollapsible>
  );
}
