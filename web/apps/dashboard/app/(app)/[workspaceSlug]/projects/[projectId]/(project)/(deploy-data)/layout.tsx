import { ProjectDataProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import type { PropsWithChildren } from "react";

// Only logs and requests read deploy data. One provider for both keeps its
// polling off the resource pages and keeps it mounted when switching between them.
export default function DeployDataLayout({ children }: PropsWithChildren) {
  return <ProjectDataProvider>{children}</ProjectDataProvider>;
}
