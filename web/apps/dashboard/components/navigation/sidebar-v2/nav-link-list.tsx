"use client";

import { SidebarGroup, SidebarGroupContent, SidebarMenu } from "@/components/ui/sidebar";
import type { ResolvedNavLink } from "@/lib/navigation/types";
import { NavRow } from "./nav-row";

export function NavLinkList({ links }: { links: ResolvedNavLink[] }) {
  return (
    <SidebarGroup>
      <SidebarGroupContent>
        <SidebarMenu>
          {links.map((link) => (
            <NavRow key={link.key} link={link} />
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
