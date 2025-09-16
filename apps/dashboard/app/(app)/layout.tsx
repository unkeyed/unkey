"use client";

import { AppSidebar } from "@/components/navigation/sidebar/app-sidebar";
import { SidebarMobile } from "@/components/navigation/sidebar/sidebar-mobile";
import { SidebarProvider } from "@/components/ui/sidebar";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Button, Empty, Loading } from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { QueryTimeProvider } from "../../providers/query-time-provider";

interface LayoutProps {
  children: React.ReactNode;
}

interface LoadingSpinnerProps {
  message?: string;
}

function LoadingSpinner({ message = "Loading workspace..." }: LoadingSpinnerProps) {
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

interface ErrorStateProps {
  title: string;
  message: string;
  onRetry?: () => void;
  retryLabel?: string;
}

function ErrorState({ title, message, onRetry, retryLabel = "Try again" }: ErrorStateProps) {
  return (
    <div className="h-[100dvh] relative flex flex-col overflow-hidden bg-white dark:bg-base-12 lg:flex-row">
      <div className="flex items-center justify-center w-full h-full">
        <Empty>
          <Empty.Icon />
          <Empty.Title>{title}</Empty.Title>
          <Empty.Description>
            {message}
            {onRetry && (
              <div className="mt-4">
                <Button onClick={onRetry} variant="outline" size="sm">
                  {retryLabel}
                </Button>
              </div>
            )}
          </Empty.Description>
        </Empty>
      </div>
    </div>
  );
}

export default function Layout({ children }: LayoutProps) {
  const router = useRouter();
  const [hasRedirected, setHasRedirected] = useState(false);
  const retryAttemptRef = useRef(0);
  const retryTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const {
    data: user,
    isLoading: userLoading,
    error: userError,
    refetch: refetchUser,
  } = trpc.user.getCurrentUser.useQuery(undefined, {
    staleTime: 1000 * 60 * 5, // 5 minutes - data is fresh
    cacheTime: 1000 * 60 * 15, // 15 minutes - keep in cache
    retry: (failureCount, error) => {
      // Don't retry on auth errors
      if (error?.data?.code === "UNAUTHORIZED" || error?.data?.code === "FORBIDDEN") {
        return false;
      }
      return failureCount < 2;
    },
    refetchOnWindowFocus: false,
    refetchOnReconnect: true,
    refetchInterval: 1000 * 60 * 10,
  });

  const {
    workspace,
    quotas,
    isLoading: workspaceLoading,
    error: workspaceError,
    refetch: refetchWorkspace,
  } = useWorkspace();

  // Auto-retry logic for workspace loading failures on first-time login
  useEffect(() => {
    if (workspaceError && !workspace && !workspaceLoading && !hasRedirected) {
      const isContextError = workspaceError.message?.includes("workspace not found in context");
      const isNotFoundError = workspaceError?.message?.includes("NOT_FOUND");

      // Only auto-retry for specific transient errors during initial load
      if ((isContextError || isNotFoundError) && retryAttemptRef.current < 2) {
        retryAttemptRef.current += 1;
        const retryDelay = Math.min(1000 * retryAttemptRef.current, 3000); // 1s, 2s max

        console.debug(
          `Auto-retrying workspace load (attempt ${retryAttemptRef.current}/2) in ${retryDelay}ms`,
        );

        retryTimeoutRef.current = setTimeout(() => {
          refetchWorkspace();
        }, retryDelay);

        return () => {
          if (retryTimeoutRef.current) {
            clearTimeout(retryTimeoutRef.current);
            retryTimeoutRef.current = null;
          }
        };
      }
    }

    // Reset retry count on successful workspace load
    if (workspace && !workspaceError) {
      retryAttemptRef.current = 0;
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current);
        retryTimeoutRef.current = null;
      }
    }
  }, [workspaceError, workspace, workspaceLoading, hasRedirected, refetchWorkspace]);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current);
      }
    };
  }, []);

  // Handle auth and workspace redirects
  useEffect(() => {
    if (hasRedirected || userLoading) {
      return;
    }

    // Handle authentication failures or missing user
    const isAuthError =
      userError?.data?.code === "UNAUTHORIZED" || userError?.data?.code === "FORBIDDEN";
    if (!user || isAuthError) {
      setHasRedirected(true);
      router.push("/auth/sign-in");
      return;
    }

    // Handle missing org/role (should create new workspace)
    if (!user.orgId || !user.role) {
      setHasRedirected(true);
      router.push("/new");
      return;
    }

    // Handle workspace not found (should create new workspace)
    // This includes cases where workspace is null due to context errors
    if (!workspaceLoading && !workspace && !workspaceError) {
      setHasRedirected(true);
      router.push("/new");
      return;
    }

    // Handle workspace context errors specifically (user needs workspace)
    // But only after we've exhausted auto-retry attempts
    if (
      workspaceError?.message?.includes("workspace not found in context") &&
      retryAttemptRef.current >= 2
    ) {
      setHasRedirected(true);
      router.push("/new");
      return;
    }
  }, [
    user,
    userLoading,
    userError,
    workspace,
    workspaceLoading,
    workspaceError,
    router,
    hasRedirected,
  ]);

  // Show loading state while fetching critical data
  if (userLoading || workspaceLoading) {
    return <LoadingSpinner message="Loading workspace..." />;
  }

  // Show error states with retry options
  const isAuthError =
    userError?.data?.code === "UNAUTHORIZED" || userError?.data?.code === "FORBIDDEN";
  if (userError && !user && !isAuthError) {
    return (
      <ErrorState
        title="Authentication Error"
        message="Failed to authenticate. Please try again or contact support if the problem persists."
        onRetry={refetchUser}
        retryLabel="Retry Authentication"
      />
    );
  }

  if (workspaceError && !workspace) {
    // Don't show error state for context errors - these are handled by redirects
    const isContextError = workspaceError.message?.includes("workspace not found in context");

    if (!isContextError) {
      return (
        <ErrorState
          title="Workspace Error"
          message="Failed to load workspace data. Please try again or contact support if the problem persists."
          onRetry={refetchWorkspace}
          retryLabel="Retry Loading"
        />
      );
    }
  }

  if (!workspace) {
    return <LoadingSpinner message="Setting up workspace..." />;
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
              <div className="fixed top-0 inset-x-0 z-50 flex justify-center  border-t-2 border-error-9">
                <div className="bg-error-9  flex -mt-1 font-mono items-center gap-2 text-white text-xs rounded-b overflow-hidden shadow-lg select-none pointer-events-none px-1.5 py-0.5">
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
