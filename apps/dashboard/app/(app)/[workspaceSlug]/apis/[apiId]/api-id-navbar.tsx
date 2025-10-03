"use client";

import { QuickNavPopover } from "@/components/navbar-popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { useIsMobile } from "@/hooks/use-mobile";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import type { Workspace } from "@unkey/db";
import { ChevronExpandY, Nodes, Plus, TaskUnchecked } from "@unkey/icons";
import { CreateKeyDialog } from "./_components/create-key";

// Types for better type safety
interface ApiLayoutData {
  currentApi: {
    id: string;
    name: string;
    workspaceId: string;
    keyAuthId: string | null;
    keyspaceDefaults: {
      prefix?: string;
      bytes?: number;
    } | null;
    deleteProtection: boolean | null;
    ipWhitelist: string | null;
  };
  workspaceApis: Array<{
    id: string;
    name: string;
  }>;
  keyAuth: {
    id: string;
    defaultPrefix: string | null;
    defaultBytes: number | null;
    sizeApprox: number;
  } | null;
  workspace: {
    id: string;
    ipWhitelist: boolean;
  };
}

interface ApisNavbarProps {
  apiId: string;
  keyspaceId?: string;
  keyId?: string;
  activePage?: {
    href: string;
    text: string;
  };
}

interface LoadingNavbarProps {
  workspace: Workspace;
}

interface NavbarContentProps {
  apiId: string;
  keyspaceId?: string;
  keyId?: string;
  activePage?: {
    href: string;
    text: string;
  };
  workspace: Workspace;
  isMobile: boolean;
  layoutData: ApiLayoutData;
}

// Loading state component
const LoadingNavbar = ({ workspace }: LoadingNavbarProps) => (
  <Navbar>
    <Navbar.Breadcrumbs icon={<Nodes />}>
      <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/apis`}>APIs</Navbar.Breadcrumbs.Link>
      <Navbar.Breadcrumbs.Link href="#" isIdentifier className="group" noop>
        <div className="h-6 w-20 bg-grayA-3 rounded animate-pulse transition-all " />
      </Navbar.Breadcrumbs.Link>
      <Navbar.Breadcrumbs.Link href="#" noop active>
        <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
          <div className="h-6 w-16 bg-grayA-3 rounded animate-pulse transition-all " />
          <ChevronExpandY className="size-4" />
        </div>
      </Navbar.Breadcrumbs.Link>
    </Navbar.Breadcrumbs>
    <Navbar.Actions>
      <NavbarActionButton disabled>
        <Plus />
        Create new key
      </NavbarActionButton>
      <div className="h-7 bg-grayA-2 border border-gray-6 rounded-md animate-pulse px-3 flex gap-2 items-center justify-center w-[190px] transition-all ">
        <div className="h-3 w-[190px] bg-grayA-3 rounded" />
        <div>
          <TaskUnchecked iconsize="md-regular" className="!size-4" />
        </div>
      </div>
    </Navbar.Actions>
  </Navbar>
);

// Main navbar content component
const NavbarContent = ({
  keyspaceId,
  keyId,
  activePage,
  workspace,
  isMobile,
  layoutData,
}: NavbarContentProps) => {
  const shouldFetchKey = Boolean(keyspaceId && keyId);

  // If we expected to find a key but this component doesn't handle key details,
  // we should handle this at a higher level or in a different component
  if (shouldFetchKey) {
    console.warn("Key fetching logic should be handled at a higher level");
  }

  const { currentApi } = layoutData;

  // Define base path for API navigation
  const base = `/${workspace.slug}/apis/${currentApi.id}`;

  // Create navigation items for QuickNavPopover
  const navigationItems = [
    {
      id: "requests",
      label: "Requests",
      href: `/${workspace.slug}/apis/${currentApi.id}`,
    },
  ];

  // Add Keys navigation if keyAuthId exists
  if (currentApi.keyAuthId) {
    navigationItems.push({
      id: "keys",
      label: "Keys",
      href: `/${workspace.slug}/apis/${currentApi.id}/keys/${currentApi.keyAuthId}`,
    });
  }

  // Add Settings navigation
  navigationItems.push({
    id: "settings",
    label: "Settings",
    href: `/${workspace.slug}/apis/${currentApi.id}/settings`,
  });

  return (
    <div className="w-full">
      <Navbar className="w-full flex justify-between">
        <Navbar.Breadcrumbs className="flex-1 w-full" icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link
            href={`/${workspace.slug}/apis`}
            className={isMobile ? "hidden" : "max-md:hidden"}
          >
            APIs
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={base}
            isIdentifier
            className={isMobile ? "hidden" : "group max-md:hidden"}
            noop
          >
            <div className="text-accent-10 group-hover:text-accent-12">{currentApi.name}</div>
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={activePage?.href ?? ""} noop active={!shouldFetchKey}>
            <QuickNavPopover items={navigationItems} shortcutKey="M">
              <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
                {activePage?.text ?? ""}
                <ChevronExpandY className="size-4" />
              </div>
            </QuickNavPopover>
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        {layoutData.keyAuth ? (
          <CreateKeyDialog
            keyspaceId={layoutData.keyAuth.id}
            apiId={currentApi.id}
            copyIdValue={currentApi.id}
            keyspaceDefaults={currentApi.keyspaceDefaults}
          />
        ) : (
          <NavbarActionButton disabled>
            <Plus />
            Create new key
          </NavbarActionButton>
        )}
      </Navbar>
    </div>
  );
};

// Main component
export const ApisNavbar = ({ apiId, keyspaceId, keyId, activePage }: ApisNavbarProps) => {
  const workspace = useWorkspaceNavigation();

  const isMobile = useIsMobile();

  // Only make the query if we have a valid apiId
  const {
    data: layoutData,
    isLoading,
    error,
  } = trpc.api.queryApiKeyDetails.useQuery(
    { apiId },
    {
      enabled: Boolean(apiId), // Only run query if apiId exists
      retry: 3,
      retryDelay: 1000,
    },
  );

  // Show loading state while fetching data
  if (isLoading || !layoutData) {
    return <LoadingNavbar workspace={workspace} />;
  }

  // Handle error state
  if (error) {
    console.error("Failed to fetch API layout data:", error);
    return <LoadingNavbar workspace={workspace} />;
  }

  return (
    <NavbarContent
      apiId={apiId}
      keyspaceId={keyspaceId}
      keyId={keyId}
      activePage={activePage}
      workspace={workspace}
      isMobile={isMobile}
      layoutData={layoutData}
    />
  );
};
