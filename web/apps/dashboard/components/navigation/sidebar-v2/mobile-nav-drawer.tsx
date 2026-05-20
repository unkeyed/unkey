"use client";

import { useSidebar } from "@/components/ui/sidebar";
import { Drawer } from "@unkey/ui";
import { usePathname } from "next/navigation";
import { useEffect } from "react";
import { SidebarBody } from "./sidebar-body";

export function MobileNavDrawer() {
  const { isMobile, openMobile, setOpenMobile } = useSidebar();
  const pathname = usePathname();

  // biome-ignore lint/correctness/useExhaustiveDependencies: pathname is the trigger
  useEffect(() => {
    setOpenMobile(false);
  }, [pathname, setOpenMobile]);

  if (!isMobile) {
    return null;
  }

  return (
    <Drawer.Root open={openMobile} onOpenChange={setOpenMobile}>
      <Drawer.Content>
        <Drawer.Title className="sr-only">Navigation</Drawer.Title>
        <Drawer.Description className="sr-only">
          Navigate to sections and sub-pages of the dashboard.
        </Drawer.Description>
        <div className="overflow-y-auto">
          <SidebarBody />
        </div>
      </Drawer.Content>
    </Drawer.Root>
  );
}
