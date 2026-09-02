"use client";

import { Fragment } from "react";
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarMenu,
  SidebarSeparator,
} from "@/components/ui/sidebar";
import type { ResolvedNavLink } from "@/lib/navigation/types";
import { NavRow } from "./nav-row";

export function NavLinkList({ links }: { links: ResolvedNavLink[] }) {
  return (
    <SidebarGroup>
      <SidebarGroupContent>
        <SidebarMenu>
          {links.map((link) => (
            <Fragment key={link.key}>
              {link.separatorAbove ? <SidebarSeparator className="-mx-2 my-1 bg-grayA-4" /> : null}
              <NavRow link={link} />
            </Fragment>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
