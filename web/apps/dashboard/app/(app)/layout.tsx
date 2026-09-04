"use client";

import { ComputePausedBanner } from "@/components/navigation/compute-paused-banner";
import { SIDEBAR_WIDTH_VARS, SidebarV2 } from "@/components/navigation/sidebar-v2";
import { MobileNavDrawer } from "@/components/navigation/sidebar-v2/mobile-nav-drawer";
import { TopNav } from "@/components/navigation/top-nav";
import { SidebarProvider } from "@/components/ui/sidebar";

import { LoadingState } from "@/components/loading-state";
import { redirectToSignIn } from "@/lib/auth/redirect-utils";
import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import { Empty } from "@unkey/ui";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import { QueryTimeProvider } from "../../providers/query-time-provider";

interface LayoutProps {
  children: React.ReactNode;
}

type WorkspaceLike = { id: string; enabled: boolean };

function WorkspaceContent({
  workspace,
  children,
}: {
  workspace: WorkspaceLike;
  children: React.ReactNode;
}) {
  return (
    <div className="w-full flex-1 flex flex-col">
      {workspace.enabled ? (
        <QueryTimeProvider>{children}</QueryTimeProvider>
      ) : (
        <div className="flex items-center justify-center w-full h-full">
          <Empty>
            <Empty.Icon />
            <Empty.Title>This workspace is disabled</Empty.Title>
            <Empty.Description>
              Contact{" "}
              <Link
                href={`mailto:support@unkey.com?body=workspaceId: ${workspace.id}`}
                className="underline"
              >
                support@unkey.com
              </Link>
            </Empty.Description>
          </Empty>
        </div>
      )}
    </div>
  );
}

export default function Layout({ children }: LayoutProps) {
  const router = useRouter();
  const pathname = usePathname();
  const { user, workspace, isLoading, error, workspaceMissing } = useWorkspace();
  // Creation wizards are focused full-screen experiences without the sidebar.
  const isCreationWizard =
    /\/projects\/[^/]+\/apps\/new$/.test(pathname) || /\/settings\/logdrains\/new$/.test(pathname);

  useEffect(() => {
    // Don't navigate while loading
    if (isLoading) {
      return;
    }

    // Handle authentication errors
    const isAuthError = error?.data?.code === "UNAUTHORIZED" || error?.data?.code === "FORBIDDEN";

    if (isAuthError) {
      redirectToSignIn(window.location);
      return;
    }

    // Handle cases where user needs workspace setup
    // Case 1: User exists but no orgId or role (incomplete setup)
    // Case 2: No workspace exists for the org (WorkOS org without workspace)
    if (user && (!user.orgId || workspaceMissing)) {
      router.push(routes.workspaces.create());
      return;
    }
  }, [user, isLoading, error, workspaceMissing, router]);

  // Show loading state while checking authentication and workspace
  if (isLoading || !user || !workspace) {
    return (
      <div className="h-dvh flex flex-col">
        <LoadingState message="Loading workspace..." />
      </div>
    );
  }

  return (
    <SidebarProvider style={SIDEBAR_WIDTH_VARS}>
      <div className="h-dvh w-full flex flex-col overflow-hidden bg-white dark:bg-base-12">
        <ComputePausedBanner />
        <TopNav />
        <MobileNavDrawer />
        <div className="relative flex flex-1 overflow-hidden">
          {!isCreationWizard && <SidebarV2 className="bg-gray-1 border-grayA-4" />}
          {/* Reserve the scrollbar gutter so content doesn't shift horizontally
              when the scrollbar appears/disappears (e.g. a dialog locking scroll
              or content height changing). Without this the centered layout
              "shakes" and buttons move out from under the cursor (ENG-2884). */}
          <div className="flex-1 overflow-auto" style={{ scrollbarGutter: "stable" }}>
            <div
              className="isolate bg-base-12 w-full min-h-full flex flex-col items-center"
              id="layout-wrapper"
            >
              <WorkspaceContent workspace={workspace}>{children}</WorkspaceContent>
            </div>
          </div>
        </div>
      </div>
    </SidebarProvider>
  );
}
