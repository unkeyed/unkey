"use client";
import { WorkspaceSwitcher } from "@/components/navigation/sidebar/team-switcher";
import { UserButton } from "@/components/navigation/sidebar/user-button";
import { useSidebar } from "@/components/ui/sidebar";
import { SidebarLeftShow } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { HelpButton } from "./help-button";

export const SidebarMobile = () => {
  const { isMobile, setOpenMobile, state, openMobile } = useSidebar();

  if (!isMobile) {
    return null;
  }

  return (
    <div className="flex w-full gap-4 py-4 pr-4 px-2 border-b border-grayA-4 items-center bg-gray-1 justify-between">
      <Button
        variant="ghost"
        onClick={() => setOpenMobile(true)}
        className="[&_svg]:size-[20px]"
      >
        <SidebarLeftShow iconsize="xl-medium" className="text-gray-9" />
      </Button>
      <WorkspaceSwitcher />
      <div className="flex gap-4 items-center">
        <HelpButton />
        <UserButton
          isCollapsed={
            (state === "collapsed" || isMobile) && !(isMobile && openMobile)
          }
          isMobile={isMobile}
          isMobileSidebarOpen={openMobile}
        />
      </div>
    </div>
  );
};
