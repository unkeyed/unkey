"use client";

import { getButtonStyles } from "@/components/navigation/sidebar/app-sidebar/components/nav-items/utils";
import { SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import type { ResolvedNavLink } from "@/lib/navigation/types";
import Link from "next/link";

// Single-row primitive. `getButtonStyles` carries the legacy active visuals
// (bg-grayA-3 / text-accent-12) because the dashboard theme doesn't define
// --sidebar-accent yet — theming the sidebar tokens is a follow-up.
//
// Disabled rows render as a real disabled <button> instead of `asChild` Link:
// aria-disabled on a Link still allows Enter to navigate. The disabled
// button removes the row from tab order and blocks activation.
export function NavRow({ link }: { link: ResolvedNavLink }) {
  const Icon = link.icon;
  const tooltip = typeof link.label === "string" ? link.label : undefined;
  const contents = (
    <>
      {Icon ? <Icon iconSize="xl-medium" /> : null}
      <span>{link.label}</span>
      {link.tag ? <div className="ml-auto">{link.tag}</div> : null}
    </>
  );

  if (link.disabled) {
    return (
      <SidebarMenuItem>
        <SidebarMenuButton
          disabled
          tooltip={tooltip}
          isActive={link.isActive}
          className={getButtonStyles(link.isActive)}
        >
          {contents}
        </SidebarMenuButton>
      </SidebarMenuItem>
    );
  }

  const linkProps = link.external ? { target: "_blank", rel: "noopener noreferrer" } : {};

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        asChild
        tooltip={tooltip}
        isActive={link.isActive}
        className={getButtonStyles(link.isActive)}
      >
        <Link href={link.href} {...linkProps}>
          {contents}
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}
