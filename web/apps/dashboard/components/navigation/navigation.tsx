"use client";

import { Navbar } from "@/components/navigation/navbar";
import type { ReactNode } from "react";

// Reusable for settings where we only change the link
export function Navigation({ href, name, icon }: { href: string; name: string; icon: ReactNode }) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={icon}>
        <Navbar.Breadcrumbs.Link href={href} active>
          {name}
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
    </Navbar>
  );
}
