"use client";

import { UpgradeSuccessDialog } from "@/components/billing/upgrade-success/upgrade-success-dialog";
import { ComputePausedBanner } from "@/components/navigation/compute-paused-banner";
import { SIDEBAR_WIDTH_VARS, SidebarV2 } from "@/components/navigation/sidebar-v2";
import { MobileNavDrawer } from "@/components/navigation/sidebar-v2/mobile-nav-drawer";
import { TopNav } from "@/components/navigation/top-nav";
import { SidebarProvider } from "@/components/ui/sidebar";

import { LoadingState } from "@/components/loading-state";
import { useWorkspaceLimits } from "@/hooks/use-workspace-limits";
import { useWorkspaceUsage } from "@/hooks/use-workspace-usage";
import { redirectToSignIn } from "@/lib/auth/redirect-utils";
import { useBillingUIUpgrades } from "@/lib/flags/use-billing-ui-upgrades";
import { routes } from "@/lib/navigation/routes";
import { useWorkspace } from "@/providers/workspace-provider";
import { EmptyState, EmptyStateDescription, EmptyStateHeader, EmptyStateTitle } from "@unkey/ui";
import Link from "next/link";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect } from "react";
import { QueryTimeProvider } from "../../providers/query-time-provider";
import { resolveUsagePeriod } from "./[workspaceSlug]/settings/usage/period";

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
    <div className="w-full flex-1 flex flex-col md:has-data-[width=full]:min-h-0 md:has-data-[width=full]:basis-0">
      {workspace.enabled ? (
        <QueryTimeProvider>{children}</QueryTimeProvider>
      ) : (
        <div className="flex flex-1 items-center justify-center p-12">
          <EmptyState>
            <EmptyStateHeader>
              <EmptyStateTitle>This workspace is disabled</EmptyStateTitle>
              <EmptyStateDescription>
                Contact{" "}
                <Link
                  href={`mailto:support@unkey.com?body=workspaceId: ${workspace.id}`}
                  className="underline"
                >
                  support@unkey.com
                </Link>
              </EmptyStateDescription>
            </EmptyStateHeader>
          </EmptyState>
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
    pathname === "/integrations/github/callback" ||
    /\/projects\/[^/]+\/apps\/new$/.test(pathname) ||
    /\/settings\/logdrains\/new$/.test(pathname) ||
    pathname === routes.cli.device();
  const billingUpgrades = useBillingUIUpgrades();
  const searchParams = useSearchParams();
  // These reads need the authenticated organization, not the workspace response.
  useWorkspaceLimits({ enabled: !isCreationWizard });
  useWorkspaceUsage(resolveUsagePeriod(searchParams.get("period")), {
    enabled: pathname.endsWith("/settings/usage") && billingUpgrades,
  });

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
        <LoadingState delayMs={300} message="Loading workspace..." />
      </div>
    );
  }

  return (
    <SidebarProvider style={SIDEBAR_WIDTH_VARS}>
      <Suspense fallback={null}>
        <UpgradeSuccessDialog />
      </Suspense>
      <div className="h-dvh w-full flex flex-col overflow-hidden bg-background">
        <ComputePausedBanner />
        <TopNav />
        <MobileNavDrawer />
        <div className="relative flex flex-1 overflow-hidden">
          {!isCreationWizard && <SidebarV2 className="bg-background" />}
          <div className="flex-1 overflow-auto">
            <div
              className="isolate bg-background w-full min-h-full flex flex-col items-center"
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
