"use client";

import { SIDEBAR_WIDTH_VARS, SidebarV2 } from "@/components/navigation/sidebar-v2";
import { MobileNavDrawer } from "@/components/navigation/sidebar-v2/mobile-nav-drawer";
import { TopNav } from "@/components/navigation/top-nav";
import { SidebarProvider } from "@/components/ui/sidebar";
import type { Route } from "next";

import { LoadingState } from "@/components/loading-state";
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

function ImpersonationBanner() {
  return (
    <div className="fixed top-0 inset-x-0 z-50 flex justify-center border-t-2 border-error-9">
      <div className="bg-error-9 flex -mt-1 font-mono items-center gap-2 text-white text-xs rounded-b overflow-hidden shadow-lg select-none pointer-events-none px-1.5 py-0.5">
        Impersonation Mode. Do not change anything and log out after you are done.
      </div>
    </div>
  );
}

export default function Layout({ children }: LayoutProps) {
  const router = useRouter();
  const pathname = usePathname();
  const { user, workspace, isLoading, error, workspaceMissing } = useWorkspace();
  // The app onboarding flow is a focused full-screen experience without the sidebar.
  const isAppOnboarding = /\/projects\/[^/]+\/apps\/new$/.test(pathname);

  useEffect(() => {
    // Don't navigate while loading
    if (isLoading) {
      return;
    }

    // Handle authentication errors
    const isAuthError = error?.data?.code === "UNAUTHORIZED" || error?.data?.code === "FORBIDDEN";

    if (isAuthError) {
      const currentPath = window.location.pathname + window.location.search;
      const signInUrl =
        currentPath && currentPath !== "/"
          ? `/auth/sign-in?redirect=${encodeURIComponent(currentPath)}`
          : "/auth/sign-in";
      router.push(signInUrl as Route);
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

  const isImpersonator = user?.impersonator;

  return (
    <SidebarProvider style={SIDEBAR_WIDTH_VARS}>
      <div className="h-dvh w-full flex flex-col overflow-hidden bg-white dark:bg-base-12">
        <TopNav />
        <MobileNavDrawer />
        <div className="flex flex-1 overflow-hidden">
          {!isAppOnboarding && <SidebarV2 className="bg-gray-1 border-grayA-4" />}
          {/* Reserve the scrollbar gutter so content doesn't shift horizontally
              when the scrollbar appears/disappears (e.g. a dialog locking scroll
              or content height changing). Without this the centered layout
              "shakes" and buttons move out from under the cursor (ENG-2884). */}
          <div className="flex-1 overflow-auto" style={{ scrollbarGutter: "stable" }}>
            <div
              className="isolate bg-base-12 w-full min-h-full overflow-x-auto flex flex-col items-center"
              id="layout-wrapper"
            >
              <WorkspaceContent workspace={workspace}>{children}</WorkspaceContent>
            </div>
            {isImpersonator ? <ImpersonationBanner /> : null}
          </div>
        </div>
      </div>
    </SidebarProvider>
  );
}
