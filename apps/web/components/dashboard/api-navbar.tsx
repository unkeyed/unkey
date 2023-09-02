"use client";

import { usePathname } from "next/navigation";

import { cn } from "@/lib/utils";
import Link from "next/link";
import React from "react";

export type NavLinkProps = {
  label: string;
  href: string;
};
export const NavLink: React.FC<NavLinkProps> = ({ label, href }) => {
  const path = usePathname();
  return (
    <Link
      href={href}
      className={cn(
        path === href
          ? "border-gray-900 text-gray-900 font-medium dark:text-gray-100 dark:border-gray-100"
          : "text-gray-600 border-transparent hover:border-gray-500 hover:text-gray-900 dark:hover:text-gray-300 dark:hover:border-gray-300",
        "border-b py-2 px-3 inline-flex duration-150 transition-all items-center text-sm ",
      )}
    >
      {label}
    </Link>
  );
};
