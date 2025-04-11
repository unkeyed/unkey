"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { CreateKeyButton } from "@/components/dashboard/create-key-button";
import { QuickNavPopover } from "@/components/navbar-popover";
import { Navbar } from "@/components/navigation/navbar";
import { Badge } from "@/components/ui/badge";
import { ChevronExpandY, Gauge } from "@unkey/icons";
import { useMediaQuery } from "usehooks-ts";

export const ApisNavbar = ({
  api,
  apis,
  activePage,
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
  activePage: {
    href: string;
    text: string;
  };
}) => {
  const isMobile = useMediaQuery("(max-width: 768px)");
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

          <Navbar.Breadcrumbs.Link href={activePage.href} noop active>
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
              shortcutKey="M"
            >
              <div className="hover:bg-gray-3 rounded-lg flex items-center gap-1 p-1">
                {activePage.text}
                <ChevronExpandY className="size-4" />
              </div>
            </QuickNavPopover>
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions className="justify-end flex flex-1">
          <Badge
            key="namespaceId"
            variant="secondary"
            className="flex justify-between gap-2 max-md:w-[120px] font-mono font-medium ph-no-capture"
          >
            <span className="truncate">{api.id}</span>
            <CopyButton value={api.id} className="flex-shrink-0" />
          </Badge>

          <CreateKeyButton apiId={api.id} keyAuthId={api.keyAuthId!} />
        </Navbar.Actions>
      </Navbar>
    </div>
  );
};
