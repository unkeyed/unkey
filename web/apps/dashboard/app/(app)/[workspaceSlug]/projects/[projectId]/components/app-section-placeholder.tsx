"use client";

import { useProjectItems } from "@/hooks/use-project-items";
import type { IconProps } from "@unkey/icons";
import type { ComponentType } from "react";

type Props = {
  projectId: string;
  appSlug: string;
  section: string;
  icon: ComponentType<IconProps>;
};

/**
 * Prototype placeholder for an app's sub-section (deployments / logs /
 * env vars / sentinel policies / settings). Blank on purpose — we're
 * exploring the nav shape, not rebuilding the pages.
 */
export function AppSectionPlaceholder({ projectId, appSlug, section, icon: Icon }: Props) {
  const { items } = useProjectItems(projectId);
  const app = items.find((i) => i.type === "app" && i.slug === appSlug);
  const appName = app?.name ?? appSlug;

  return (
    <div className="flex flex-col gap-2 p-6 w-full max-w-[1200px] mx-auto">
      <div className="flex items-center gap-3">
        <div className="size-10 bg-gray-3 rounded-[10px] flex items-center justify-center shrink-0 shadow-sm shadow-grayA-8/20 dark:ring-1 dark:ring-gray-4 dark:shadow-none">
          <Icon iconSize="xl-medium" className="size-5" />
        </div>
        <div className="flex flex-col">
          <span className="text-xs text-gray-11">{appName}</span>
          <h1 className="text-xl font-semibold text-accent-12">{section}</h1>
        </div>
      </div>
      <p className="text-sm text-gray-11 mt-2">
        Placeholder page for this app's {section.toLowerCase()}. Real content lands when the backend
        exposes apps as first-class.
      </p>
    </div>
  );
}
