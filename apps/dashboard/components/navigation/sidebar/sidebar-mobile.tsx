"use client";
import { WorkspaceSwitcher } from "@/app/(app)/team-switcher";
import { UserButton } from "@/app/(app)/user-button";
import { useSidebar } from "@/components/ui/sidebar";
import type { Workspace } from "@unkey/db";
import { Menu } from "@unkey/icons";
import { Button } from "@unkey/ui";

export const SidebarMobile = ({ workspace }: { workspace: Workspace }) => {
  const { isMobile, setOpenMobile } = useSidebar();

  if (!isMobile) {
    return null;
  }

  return (
    <div className="flex w-full gap-4 py-4 pr-4 px-2 border-b border-grayA-4 items-center bg-gray-1 justify-between">
      <Button variant="ghost" onClick={() => setOpenMobile(true)}>
        <Menu size="lg-regular" className="text-gray-9" />
      </Button>
      <WorkspaceSwitcher workspace={workspace} />
      <div className="flex gap-4 items-center">
        <UserButton />
        {/* TODO: Will be used in the next iteration as an indicator for   */}
        {/* <CircleQuestion size="xl-regular" className="text-gray-9" /> */}
      </div>
    </div>
  );
};
