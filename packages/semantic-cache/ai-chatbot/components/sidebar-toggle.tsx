"use client";

import * as React from "react";

import { Button } from "@/components/ui/button";
import { IconSidebar } from "@/components/ui/icons";
import { useSidebar } from "@/lib/hooks/use-sidebar";

export function SidebarToggle() {
  const { toggleSidebar } = useSidebar();

  return (
    <Button
      variant="ghost"
      className="-ml-2 hidden size-9 p-0 lg:flex"
      onClick={() => {
        toggleSidebar();
      }}
    >
      <IconSidebar className="size-6" />
      <span className="sr-only">Toggle Sidebar</span>
    </Button>
  );
}
