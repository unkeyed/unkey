"use client";

import { AppSidebar } from "@/components/navigation/sidebar/app-sidebar";
import { SidebarMobile } from "@/components/navigation/sidebar/sidebar-mobile";
import { SidebarProvider } from "@/components/ui/sidebar";

import { useWorkspace } from "@/providers/workspace-provider";
import { Empty, Loading } from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { QueryTimeProvider } from "../../providers/query-time-provider";

interface LayoutProps {
  children: React.ReactNode;
}

function LoadingState({ message = "Loading..." }: { message?: string }) {
  return (
    <div className="h-[100dvh] relative flex flex-col overflow-hidden bg-white dark:bg-base-12 lg:flex-row">
      <div className="flex items-center justify-center w-full h-full">
        <div className="flex flex-col items-center gap-4">
          <Loading size={24} />
          <p className="text-sm text-gray-600 dark:text-gray-400">{message}</p>
        </div>
      </div>
    </div>
  );
}

export default function Layout({ children }: LayoutProps) {
  const router = useRouter();
  const { user, workspace, quotas, isLoading, error } = useWorkspace();
  useEffect(() => {
    // Don't navigate while loading
    if (isLoading) {
      return;
    }

    // Handle authentication errors
    const isAuthError = error?.data?.code === "UNAUTHORIZED" || error?.data?.code === "FORBIDDEN";

    if (isAuthError) {
      router.push("/auth/sign-in");
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
    return <LoadingState message="Loading workspace..." />;
  }

  // Combine workspace with quotas for AppSidebar
  const workspaceWithQuotas = {
    ...workspace,
    quotas,
  };

  const isImpersonator = user?.impersonator;

  return (
    <div className="h-[100dvh] relative flex flex-col overflow-hidden bg-white dark:bg-base-12 lg:flex-row">
      <SidebarProvider>
        <div className="flex flex-1 overflow-hidden">
          {/* Desktop Sidebar */}
          <AppSidebar workspace={workspaceWithQuotas} className="bg-gray-1 border-grayA-4" />

          {/* Main content area */}
          <div className="flex-1 overflow-auto">
            <div
              className="isolate bg-base-12 w-full overflow-x-auto flex flex-col items-center"
              id="layout-wrapper"
            >
              {/* Mobile sidebar at the top of content */}
              <SidebarMobile />

              <div className="w-full">
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
            </div>
            {isImpersonator ? (
              <div className="fixed top-0 inset-x-0 z-50 flex justify-center border-t-2 border-error-9">
                <div className="bg-error-9 flex -mt-1 font-mono items-center gap-2 text-white text-xs rounded-b overflow-hidden shadow-lg select-none pointer-events-none px-1.5 py-0.5">
                  Impersonation Mode. Do not change anything and log out after you are done.
                </div>
              </div>
            ) : null}
          </div>
        </div>
      </SidebarProvider>
    </div>
  );
}
