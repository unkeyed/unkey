import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navigation } from "@/components/navigation/navigation";
import { PageContent } from "@/components/page-content";
import { Gear } from "@unkey/icons";
import type { PropsWithChildren } from "react";
import { navigation } from "../../constants";

export const Shell: React.FC<PropsWithChildren> = ({ children }) => {
  return (
    <div>
      <Navigation href="/settings/billing" name="Settings" icon={<Gear />} />
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
