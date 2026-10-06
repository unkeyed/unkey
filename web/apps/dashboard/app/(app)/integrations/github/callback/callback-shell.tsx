"use client";

import { newAppReturnProject } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/new/github-connecting";
import { Logomark } from "@/components/logomark";
import { TOP_NAV_HEIGHT, TopNav } from "@/components/navigation/top-nav";
import { CrumbSeparator } from "@/components/navigation/top-nav/crumb";
import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import { Skeleton } from "@unkey/ui";
import { useSearchParams } from "next/navigation";
import type { ReactNode } from "react";

export function CallbackShell({ children }: { children: ReactNode }) {
  const { user, workspace } = useWorkspace();
  const projectId = newAppReturnProject(useSearchParams().get("state"));
  return (
    <>
      {user && workspace ? (
        <TopNav
          crumbs={
            projectId
              ? [
                  {
                    type: "workspace",
                    href: routes.projects.list({ workspaceSlug: workspace.slug }),
                  },
                  { type: "project", owner: { state: "resolved", projectId } },
                ]
              : undefined
          }
        />
      ) : (
        <header
          className="flex w-full shrink-0 items-center gap-1 border-b bg-background px-4"
          style={{ height: TOP_NAV_HEIGHT }}
        >
          <Logomark />
          {(projectId ? ["workspace", "project"] : ["workspace"]).map((key) => (
            <span key={key} className="flex items-center gap-1">
              <CrumbSeparator />
              <span className="flex items-center gap-1.5 px-1 py-1">
                <Skeleton className="size-4 rounded bg-gray-4" />
                <Skeleton className="h-3 w-20 bg-gray-4" />
              </span>
            </span>
          ))}
        </header>
      )}
      <div
        className="flex flex-1 flex-col items-center overflow-auto"
        style={{ scrollbarGutter: "stable" }}
      >
        {children}
      </div>
    </>
  );
}
