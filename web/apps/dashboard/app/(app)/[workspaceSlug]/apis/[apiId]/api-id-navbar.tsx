"use client";

import { NavbarActionButton } from "@/components/navigation/action-button";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";
import type { Workspace } from "@unkey/db";
import { ChevronExpandY, Gear, Nodes, Plus, TaskUnchecked } from "@unkey/icons";
import { useIsMobile } from "@unkey/ui";
import { CreateKeyDialog } from "./_components/create-key";
import { KeySettingsDialog } from "./_components/key-settings-dialog";

interface ApiLayoutData {
  currentApi: {
    id: string;
    name: string;
    keyAuthId: string | null;
    keyspaceDefaults: {
      prefix?: string;
      bytes?: number;
    } | null;
  };
  keyAuth: {
    id: string;
  } | null;
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
  keyspaceId?: string;
  keyId?: string;
  activePage?: {
    href: string;
    text: string;
  };
  workspace: Workspace;
  isMobile: boolean;
  layoutData: ApiLayoutData;
  proxiedApiName?: string;
}

// Loading state component
const LoadingNavbar = ({ workspace }: LoadingNavbarProps) => (
  <Navbar>
    <Navbar.Breadcrumbs icon={<Nodes />}>
      <Navbar.Breadcrumbs.Link href={routes.apis.list({ workspaceSlug: workspace.slug })}>
        Keyspaces (APIs)
      </Navbar.Breadcrumbs.Link>
      <Navbar.Breadcrumbs.Link href="#" className="group" noop>
        <div className="h-6 w-20 bg-grayA-3 rounded-sm animate-pulse transition-all " />
      </Navbar.Breadcrumbs.Link>
      <Navbar.Breadcrumbs.Link href="#" noop active>
        <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
          <div className="h-6 w-16 bg-grayA-3 rounded-sm animate-pulse transition-all " />
          <ChevronExpandY className="size-4" />
        </div>
      </Navbar.Breadcrumbs.Link>
    </Navbar.Breadcrumbs>
    <Navbar.Actions>
      <div className="h-7 bg-grayA-2 border border-gray-6 rounded-md animate-pulse px-3 flex gap-2 items-center justify-center w-[190px] transition-all ">
        <div className="h-3 w-[190px] bg-grayA-3 rounded-sm" />
        <div>
          <TaskUnchecked iconSize="md-regular" className="size-4!" />
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
  isMobile = false,
  layoutData,
  proxiedApiName,
}: NavbarContentProps) => {
  const shouldFetchKey = Boolean(keyspaceId && keyId);

  // Fetch key details when viewing a specific key
  const {
    data: keyData,
    isLoading: isKeyLoading,
    error: keyError,
  } = trpc.api.keys.list.useQuery(
    {
      // This cannot be empty string but required to silence TS errors
      keyAuthId: keyspaceId ?? "",
      // This cannot be empty string but required to silence TS errors
      keyIds: [{ operator: "is", value: keyId ?? "" }],
      identities: null,
      limit: 1,
      names: null,
    },
    {
      enabled: shouldFetchKey,
    },
  );

  if (keyError) {
    throw new Error(`Failed to fetch key details: ${keyError.message}`);
  }

  const specificKey = keyData?.keys.find((key) => key.id === keyId);
  const currentApi = {
    ...layoutData.currentApi,
    name: proxiedApiName ?? layoutData.currentApi.name,
  };

  const base = routes.apis.detail({ workspaceSlug: workspace.slug, apiId: currentApi.id });

  return (
    <div className="w-full">
      <Navbar className="w-full flex justify-between">
        <Navbar.Breadcrumbs className="flex-1 w-full" icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link
            href={routes.apis.list({ workspaceSlug: workspace.slug })}
            className={isMobile ? "hidden" : "max-md:hidden"}
          >
            Keyspaces (APIs)
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link
            href={base}
            className={isMobile ? "hidden" : "group max-md:hidden"}
            noop
          >
            <div className="text-accent-10 group-hover:text-accent-12">{currentApi.name}</div>
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={activePage?.href ?? ""} noop active={!shouldFetchKey}>
            {activePage?.text ?? ""}
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          {shouldFetchKey && specificKey ? (
            <>
              <KeySettingsDialog
                keyData={specificKey}
                apiId={currentApi.id}
                keyspaceId={keyspaceId ?? currentApi.keyAuthId}
              />
              <CopyableIDButton value={keyId as string} />
            </>
          ) : shouldFetchKey && isKeyLoading ? (
            <>
              <NavbarActionButton disabled>
                <Gear />
                Settings
              </NavbarActionButton>
              <CopyableIDButton value={keyId as string} />
            </>
          ) : layoutData.keyAuth ? (
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
        </Navbar.Actions>
      </Navbar>
    </div>
  );
};

// Main component
export const ApisNavbar = ({ apiId, keyspaceId, keyId, activePage }: ApisNavbarProps) => {
  const workspace = useWorkspaceNavigation();

  // Default to false (desktop) to prevent hydration mismatches
  const isMobile = useIsMobile({ defaultValue: false });

  // Only make the query if we have a valid apiId
  const { data: layoutData, isLoading } = trpc.api.queryApiKeyDetails.useQuery(
    { apiId },
    {
      enabled: Boolean(apiId), // Only run query if apiId exists
      retry: 3,
      retryDelay: 1000,
    },
  );
  const { data: proxiedApi } = useQuery({
    queryKey: ["dashboard-api-proxy", "apis.getApi", apiId],
    enabled: Boolean(apiId),
    queryFn: async () => {
      const response = await getUnkeyClient().apis.getApi({ apiId });
      return response.data;
    },
  });

  // Show loading state while fetching data
  if (isLoading || !layoutData) {
    return <LoadingNavbar workspace={workspace} />;
  }

  return (
    <NavbarContent
      keyspaceId={keyspaceId}
      keyId={keyId}
      activePage={activePage}
      workspace={workspace}
      isMobile={isMobile}
      layoutData={layoutData}
      proxiedApiName={proxiedApi?.name}
    />
  );
};
