"use client";

import { SIDEBAR_WIDTH_VARS, SidebarV2 } from "@/components/navigation/sidebar-v2";
import { MobileNavDrawer } from "@/components/navigation/sidebar-v2/mobile-nav-drawer";
import { AppSidebar } from "@/components/navigation/sidebar/app-sidebar";
import { SidebarMobile } from "@/components/navigation/sidebar/sidebar-mobile";
import { TopNav } from "@/components/navigation/top-nav";
import { SidebarProvider } from "@/components/ui/sidebar";

import { LoadingState } from "@/components/loading-state";
import { useFlag } from "@/lib/flags/provider";
import { useWorkspace } from "@/providers/workspace-provider";
import { Empty } from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
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
  const { user, workspace, quotas, isLoading, error } = useWorkspace();
  const newNavigation = useFlag("newNavigation");
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
      router.push(signInUrl);
      return;
    }

    // Handle workspace not found errors - redirect to setup
    const isWorkspaceNotFound = error?.data?.code === "NOT_FOUND";

    // Handle cases where user needs workspace setup
    // Case 1: User exists but no orgId or role (incomplete setup)
    // Case 2: Workspace not found error (WorkOS org without workspace, or no organization)
    if (user && (!user.orgId || isWorkspaceNotFound)) {
      router.push("/new");
      return;
    }
  }, [user, isLoading, error, router]);

  // Show loading state while checking authentication and workspace
  if (isLoading || !user || !workspace) {
    return (
      <div className="h-dvh flex flex-col">
        <LoadingState message="Loading workspace..." />
      </div>
    );
  }

  // Combine workspace with quotas for AppSidebar
  const workspaceWithQuotas = {
    ...workspace,
    quotas,
  };

  const isImpersonator = user?.impersonator;

  if (newNavigation) {
    return (
      <SidebarProvider style={SIDEBAR_WIDTH_VARS}>
        <div className="h-dvh w-full flex flex-col overflow-hidden bg-background">
          <TopNav />
          <MobileNavDrawer />
          <div className="flex flex-1 overflow-hidden">
            <SidebarV2 className="bg-gray-1 border-grayA-4" />
            <div className="flex-1 overflow-auto">
              <div
                className="isolate bg-background w-full min-h-full overflow-x-auto flex flex-col items-center"
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

  return (
    <div className="h-dvh relative flex flex-col overflow-hidden bg-background lg:flex-row">
      <SidebarProvider>
        <div className="flex flex-1 overflow-hidden">
          {/* Desktop Sidebar */}
          <AppSidebar workspace={workspaceWithQuotas} className="bg-gray-1 border-grayA-4" />

          {/* Main content area */}
          <div className="flex-1 overflow-auto">
            <div
              className="isolate bg-background w-full min-h-full overflow-x-auto flex flex-col items-center"
              id="layout-wrapper"
            >
              {/* Mobile sidebar at the top of content */}
              <SidebarMobile />

              <WorkspaceContent workspace={workspace}>{children}</WorkspaceContent>
            </div>
            {isImpersonator ? <ImpersonationBanner /> : null}
          </div>
        </div>
      </SidebarProvider>
    </div>
  );
}
