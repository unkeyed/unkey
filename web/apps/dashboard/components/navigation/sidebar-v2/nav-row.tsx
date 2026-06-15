"use client";

import { getButtonStyles } from "@/components/navigation/sidebar/app-sidebar/components/nav-items/utils";
import { SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import type { ResolvedNavLink } from "@/lib/navigation/types";
import type { Route } from "next";
import Link from "next/link";

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
        <Link href={link.href as Route} {...linkProps}>
          {contents}
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}
