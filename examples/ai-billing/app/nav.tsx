"use client";

import { usePathname } from "next/navigation";

import { cn } from "@/lib/utils";
import Link from "next/link";
import React from "react";

export type NavLinkProps = {
  label: string;
  href: string;
  target?: string;
};

// TODO convert to function
export const NavLink: React.FC<NavLinkProps> = ({ label, href, target }) => {
  const path = usePathname();
  return (
    <Link
      href={href}
      target={target}
      className={cn(
        path === href
          ? "transition-colors hover:text-primary"
          : "text-muted-foreground transition-colors hover:text-primary",
        "text-sm font-medium",
      )}
    >
      {label}
    </Link>
  );
};
