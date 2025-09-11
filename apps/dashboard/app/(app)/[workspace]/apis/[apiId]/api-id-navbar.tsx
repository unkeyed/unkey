"use client";

import { QuickNavPopover } from "@/components/navbar-popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import { useIsMobile } from "@/hooks/use-mobile";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { ChevronExpandY, Gear, Nodes, Plus, TaskUnchecked } from "@unkey/icons";
import dynamic from "next/dynamic";
import { navigation } from "./constants";
import { getKeysTableActionItems } from "./keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover.constants";

const CreateKeyDialog = dynamic(
  () =>
    import("./_components/create-key").then((mod) => ({
      default: mod.CreateKeyDialog,
    })),
  {
    ssr: false,
  },
);

const KeysTableActionPopover = dynamic(
  () =>
    import("@/components/logs/table-action.popover").then((mod) => ({
      default: mod.TableActionPopover,
    })),
  {
    ssr: false,
  },
);

export const ApisNavbar = ({
  apiId,
  keyspaceId,
  keyId,
  activePage,
}: {
  apiId: string;
  keyspaceId?: string;
  keyId?: string;
  activePage?: {
    href: string;
    text: string;
  };
}) => {
  const { workspace } = useWorkspace();

  const isMobile = useIsMobile();
  const trpcUtils = trpc.useUtils();
  const { data: layoutData, isLoading } = trpc.api.queryApiKeyDetails.useQuery({
    apiId,
  });
  // Only fetch key data when we have keyspaceId and keyId
  const shouldFetchKey = Boolean(keyspaceId && keyId);
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
      cursor: null,
      identities: null,
      limit: 1,
      names: null,
    },
    {
      enabled: shouldFetchKey,
    },
  );

  // Handle key error
  if (keyError) {
    throw new Error(`Failed to fetch key details: ${keyError.message}`);
  }

  // Extract the specific key from the response
  const specificKey = keyData?.keys.find((key) => key.id === keyId);
  if (!layoutData || isLoading || (shouldFetchKey && isKeyLoading)) {
    return (
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace?.slug}/apis`}>APIs</Navbar.Breadcrumbs.Link>
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
          {specificKey?.id ? (
            <NavbarActionButton disabled>
              <Gear size="sm-regular" />
              Settings
            </NavbarActionButton>
          ) : (
            <NavbarActionButton disabled>
              <Plus />
              Create new key
            </NavbarActionButton>
          )}
          <div className="h-7 bg-grayA-2 border border-gray-6 rounded-md animate-pulse px-3 flex gap-2 items-center justify-center w-[190px] transition-all ">
            <div className="h-3 w-[190px] bg-grayA-3 rounded" />
            <div>
              <TaskUnchecked size="md-regular" className="!size-4" />
            </div>
          </div>
        </Navbar.Actions>
      </Navbar>
    );
  }

  // If we expected to find a key but didn't, throw an error
  if (shouldFetchKey && !specificKey) {
    throw new Error(`Key ${keyId} not found`);
  }

  const { currentApi, workspaceApis } = layoutData;

  // Define base path for API navigation
  const base = `/${workspace?.slug}/apis/${currentApi.id}`;
  const navItems = navigation(currentApi.id, currentApi.keyAuthId ?? "");

  return (
    <>
      <div className="w-full">
        <Navbar className="w-full flex justify-between">
          <Navbar.Breadcrumbs className="flex-1 w-full" icon={<Nodes />}>
            {!isMobile && (
              <>
                <Navbar.Breadcrumbs.Link
                  href={`/${workspace?.slug}/apis`}
                  className="max-md:hidden"
                >
                  APIs
                </Navbar.Breadcrumbs.Link>
                <Navbar.Breadcrumbs.Link
                  href={base}
                  isIdentifier
                  className="group max-md:hidden"
                  noop
                >
                  <QuickNavPopover
                    items={workspaceApis.map((api) => ({
                      id: api.id,
                      label: api.name,
                      href: `/${workspace?.slug}/apis/${api.id}`,
                    }))}
                    shortcutKey="N"
                  >
                    <div className="text-accent-10 group-hover:text-accent-12">
                      {currentApi.name}
                    </div>
                  </QuickNavPopover>
                </Navbar.Breadcrumbs.Link>
              </>
            )}
            <Navbar.Breadcrumbs.Link href={activePage?.href ?? ""} noop active={!specificKey}>
              <QuickNavPopover
                items={navItems.map((item) => ({
                  id: item.segment,
                  label: item.label,
                  href: item.href,
                }))}
              >
                <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
                  {activePage?.text ?? ""}
                  <ChevronExpandY className="size-4" />
                </div>
              </QuickNavPopover>
            </Navbar.Breadcrumbs.Link>
            {specificKey && (
              <Navbar.Breadcrumbs.Link
                href={`${base}/keys/${currentApi.keyAuthId}/${specificKey.id}`}
                className="max-md:hidden"
                isLast
                isIdentifier
                active
              >
                {shortenId(specificKey.id)}
              </Navbar.Breadcrumbs.Link>
            )}
          </Navbar.Breadcrumbs>
          {specificKey?.id ? (
            <div className="flex gap-3 items-center">
              <Navbar.Actions>
                <KeysTableActionPopover items={getKeysTableActionItems(specificKey, trpcUtils)}>
                  <NavbarActionButton>
                    <Gear size="sm-regular" />
                    Settings
                  </NavbarActionButton>
                </KeysTableActionPopover>
                <CopyableIDButton value={specificKey.id} />
              </Navbar.Actions>
            </div>
          ) : (
            currentApi.keyAuthId && (
              <CreateKeyDialog
                keyspaceId={currentApi.keyAuthId}
                apiId={currentApi.id}
                keyspaceDefaults={currentApi.keyspaceDefaults}
              />
            )
          )}
        </Navbar>
      </div>
    </>
  );
};
