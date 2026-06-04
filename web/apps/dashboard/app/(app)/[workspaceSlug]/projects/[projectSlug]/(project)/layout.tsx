import { ProjectDataProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectSlug]/apps/[appSlug]/(overview)/data-provider";
import type { PropsWithChildren } from "react";

export default function ProjectScopedLayout({ children }: PropsWithChildren) {
  return (
    <ProjectDataProvider>
      <div className="h-full flex flex-col overflow-hidden">
        <div className="flex flex-1 min-h-0">
          <div className="flex-1 overflow-auto">{children}</div>
        </div>
      </div>
    </ProjectDataProvider>
  );
}
