"use client";

import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { useIsMobile } from "@/hooks/use-mobile";
import { ChevronExpandY, Gauge } from "@unkey/icons";
import { CreateKeyDialog } from "./_components/create-key";

export const ApisNavbar = ({
  api,
  apis,
  activePage,
  keyId,
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
  keyId?: string;
}) => {
  const isMobile = useIsMobile();
  return (
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

          <Navbar.Breadcrumbs.Link href={activePage?.href ?? ""} noop active={!keyId}>
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
          {keyId && (
            <Navbar.Breadcrumbs.Link
              href={`/apis/${api.id}/keys/${api.keyAuthId}/${keyId}`}
              className="max-md:hidden"
              isLast
              isIdentifier
              active
            >
              {keyId?.substring(0, 8)}...{keyId?.substring(keyId?.length - 4)}
            </Navbar.Breadcrumbs.Link>
          )}
        </Navbar.Breadcrumbs>
        <CreateKeyDialog keyspaceId={api.keyAuthId} apiId={api.id} />
      </Navbar>
    </div>
  );
};
