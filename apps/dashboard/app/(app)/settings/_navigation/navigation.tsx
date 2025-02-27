"use client";

import { Navbar } from "@/components/navbar";
import { Gear } from "@unkey/icons";


// Reusable for settings where we only change the link
export function Navigation({href}: {href: string}) {
  return (
    <Navbar>
        <Navbar.Breadcrumbs icon={<Gear />}>
          <Navbar.Breadcrumbs.Link href={href} active>
            Settings
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
  );
}