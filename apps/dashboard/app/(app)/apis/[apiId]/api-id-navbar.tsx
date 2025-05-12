"use client";

import { QuickNavPopover } from "@/components/navbar-popover";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import { useIsMobile } from "@/hooks/use-mobile";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { ChevronExpandY, Gauge, Gear } from "@unkey/icons";
import { CreateKeyDialog } from "./_components/create-key";
import { KeysTableActionPopover } from "./keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { getKeysTableActionItems } from "./keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover.constants";

export const ApisNavbar = ({
  api,
  apis,
  activePage,
  keyData,
}: {
  api: {
    id: string;
    name: string;
    keyAuthId: string | null;
  };
  apis: {
    id: string;
    name: string;
  }[];
  activePage?: {
    href: string;
    text: string;
  };
  keyData?: KeyDetails | null;
}) => {
  const isMobile = useIsMobile();
  return (
    <>
      <div className="w-full">
        <Navbar className="w-full flex justify-between">
          <Navbar.Breadcrumbs className="flex-1 w-full" icon={<Gauge />}>
            {!isMobile && (
              <>
                <Navbar.Breadcrumbs.Link href="/apis" className="max-md:hidden">
                  APIs
                </Navbar.Breadcrumbs.Link>

                <Navbar.Breadcrumbs.Link
                  href={`/apis/${api.id}`}
                  isIdentifier
                  className="group max-md:hidden"
                  noop
                >
                  <QuickNavPopover
                    items={apis.map((api) => ({
                      id: api.id,
                      label: api.name,
                      href: `/apis/${api.id}`,
                    }))}
                    shortcutKey="N"
                  >
                    <div className="text-accent-10 group-hover:text-accent-12">{api.name}</div>
                  </QuickNavPopover>
                </Navbar.Breadcrumbs.Link>
              </>
            )}

            <Navbar.Breadcrumbs.Link href={activePage?.href ?? ""} noop active={!keyData}>
              <QuickNavPopover
                items={[
                  {
                    id: "requests",
                    label: "Requests",
                    href: `/apis/${api.id}`,
                  },
                  {
                    id: "keys",
                    label: "Keys",
                    href: `/apis/${api.id}/keys/${api.keyAuthId}`,
                  },
                  {
                    id: "settings",
                    label: "Settings",
                    href: `/apis/${api.id}/settings`,
                  },
                ]}
              >
                <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
                  {activePage?.text ?? ""}
                  <ChevronExpandY className="size-4" />
                </div>
              </QuickNavPopover>
            </Navbar.Breadcrumbs.Link>
            {keyData && (
              <Navbar.Breadcrumbs.Link
                href={`/apis/${api.id}/keys/${api.keyAuthId}/${keyData.id}`}
                className="max-md:hidden"
                isLast
                isIdentifier
                active
              >
                {keyData.id?.substring(0, 8)}...
                {keyData.id?.substring(keyData.id?.length - 4)}
              </Navbar.Breadcrumbs.Link>
            )}
          </Navbar.Breadcrumbs>
          {keyData?.id ? (
            <Navbar.Actions>
              <KeysTableActionPopover items={getKeysTableActionItems(keyData)}>
                <NavbarActionButton>
                  <Gear size="sm-regular" />
                  Settings
                </NavbarActionButton>
              </KeysTableActionPopover>
              <CopyableIDButton value={keyData.id} />
            </Navbar.Actions>
          ) : (
            <CreateKeyDialog keyspaceId={api.keyAuthId} apiId={api.id} />
          )}
        </Navbar>
      </div>
    </>
  );
};
