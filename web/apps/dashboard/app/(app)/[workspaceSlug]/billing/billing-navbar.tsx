"use client";

import { QuickNavPopover } from "@/components/navbar-popover";
import { CopyableIDButton } from "@/components/navigation/copyable-id-button";
import { Navbar } from "@/components/navigation/navbar";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ChevronExpandY, Coins } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";

const billingNavItems = [
  {
    id: "connect",
    href: "connect",
    text: "Stripe Connect",
  },
  {
    id: "pricing-models",
    href: "pricing-models",
    text: "Pricing Models",
  },
  {
    id: "end-users",
    href: "end-users",
    text: "End Users",
  },
  {
    id: "invoices",
    href: "invoices",
    text: "Invoices",
  },
  {
    id: "analytics",
    href: "analytics",
    text: "Analytics",
  },
];

export const BillingNavbar = ({
  activePage,
}: {
  activePage: {
    href: string;
    text: string;
  };
}) => {
  const workspace = useWorkspaceNavigation();

  return (
    <div className="flex flex-col w-full h-full">
      <Navbar>
        <Navbar.Breadcrumbs icon={<Coins />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace.slug}/billing`}>
            Customer Billing
          </Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href={activePage.href} noop active>
            <QuickNavPopover
              items={billingNavItems.flatMap((item) => [
                {
                  id: item.href,
                  label: item.text,
                  href: `/${workspace.slug}/billing/${item.href}`,
                },
              ])}
              shortcutKey="B"
            >
              <div className="flex items-center gap-1 p-1 rounded-lg hover:bg-gray-3">
                {activePage.text}
                <ChevronExpandY className="size-4" />
              </div>
            </QuickNavPopover>
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          {activePage.href === "connect" && workspace && <CopyableIDButton value={workspace.id} />}
          {activePage.href === "analytics" && (
            <Link href={`/${workspace.slug}/billing/analytics?export=true`}>
              <Button type="button" variant="outline">
                Export Data
              </Button>
            </Link>
          )}
        </Navbar.Actions>
      </Navbar>
    </div>
  );
};
