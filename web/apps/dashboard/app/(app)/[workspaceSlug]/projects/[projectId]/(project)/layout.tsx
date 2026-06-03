import { ProjectDataProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import type { PropsWithChildren } from "react";

// Project-scoped routes (Logs, Requests, Settings) live outside apps/[appId], so
// they mount their own ProjectDataProvider with no appId: queries span the whole
// project.
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
