import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navbar } from "@/components/navigation/navbar";
import { PageContent } from "@/components/page-content";
import { Gear } from "@unkey/icons";
import { Button } from "@unkey/ui";
import Link from "next/link";
import type { PropsWithChildren } from "react";
import { navigation } from "../../constants";

export const Shell: React.FC<PropsWithChildren> = ({ children }) => {
  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gear />}>
          <Navbar.Breadcrumbs.Link href="/settings">Settings</Navbar.Breadcrumbs.Link>
          <Navbar.Breadcrumbs.Link href="/settings/billing" active>
            Billing
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <Button variant="outline">
            <Link href="https://cal.com/james-r-perkins/sales" target="_blank">
              Schedule a call
            </Link>
          </Button>
          <Button variant="primary">
            <Link href="mailto:support@unkey.dev">Contact us</Link>
          </Button>
        </Navbar.Actions>
      </Navbar>
      <PageContent>
        <SubMenu navigation={navigation} segment="billing" />
        <div className="py-3 w-full flex items-center justify-center ">
          <div className="w-[760px] mt-4 flex flex-col justify-center items-center gap-5">
            <h1 className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
              Billing Settings
            </h1>
            {children}
          </div>
        </div>
      </PageContent>
    </div>
  );
};
