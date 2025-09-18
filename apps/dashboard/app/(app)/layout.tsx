"use client";

import { AppSidebar } from "@/components/navigation/sidebar/app-sidebar";
import { SidebarMobile } from "@/components/navigation/sidebar/sidebar-mobile";
import { SidebarProvider } from "@/components/ui/sidebar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useWorkspace } from "@/providers/workspace-provider";
import { Empty, Loading } from "@unkey/ui";
import Link from "next/link";
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
  const { user, workspace, quotas } = useWorkspace();
  const { isReady } = useWorkspaceNavigation();

  // Show loading state
  if (!isReady || !workspace || !user) {
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
                          href={`mailto:support@unkey.dev?body=workspaceId: ${workspace.id}`}
                          className="underline"
                        >
                          support@unkey.dev
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
