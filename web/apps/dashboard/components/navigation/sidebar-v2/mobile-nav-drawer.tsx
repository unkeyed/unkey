"use client";

import { useSidebar } from "@/components/ui/sidebar";
import { Drawer } from "@unkey/ui";
import { usePathname } from "next/navigation";
import { useEffect } from "react";
import { SidebarBody } from "./sidebar-body";

// Bottom drawer surfacing SidebarBody on mobile. Built on vaul (via
// @unkey/ui's Drawer) rather than the local Sheet primitive — vaul is
// purpose-built for mobile bottom sheets: pinned to the bottom,
// drag-to-dismiss, momentum, max-h baked in. Reuses shadcn's
// SidebarProvider openMobile state so the hamburger in TopNav drives
// this drawer.
export function MobileNavDrawer() {
  const { isMobile, openMobile, setOpenMobile } = useSidebar();
  const pathname = usePathname();

  // biome-ignore lint/correctness/useExhaustiveDependencies: pathname is the trigger; the effect body intentionally closes the drawer on route change.
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
