"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import Link from "next/link";
import { useAppId, useProjectData } from "../../../data-provider";

export function OpenApiFields() {
  const workspace = useWorkspaceNavigation();
  const { projectId } = useProjectData();
  const appId = useAppId();

  return (
    <p className="text-sm leading-5 text-gray-11">
      {"Uses the auto-scraped spec. "}
      <Link
        href={routes.projects.apps.settings({
          workspaceSlug: workspace.slug,
          projectId,
          appId,
          page: "advanced",
        })}
        className="font-medium text-gray-12 underline decoration-dotted underline-offset-3"
      >
        Configure scrape path
      </Link>
    </p>
  );
}
