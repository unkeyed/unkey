import { ProjectDataProvider } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import type { PropsWithChildren } from "react";

// Only logs and requests read deploy data. Mounting the provider here rather
// than on the project layout keeps its deployment, app, domain, environment and
// custom-domain subscriptions (and their polling) off the keyspace, ratelimit,
// identity, authorization and settings pages.
export function DeployDataLayout({ children }: PropsWithChildren) {
  return <ProjectDataProvider>{children}</ProjectDataProvider>;
}
