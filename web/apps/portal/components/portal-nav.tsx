"use client";

import { deriveVisibleTabs } from "@/lib/permissions";
import { usePathname } from "next/navigation";
import Link from "next/link";

export function PortalNav({
  permissions,
  logoUrl,
}: {
  permissions: ReadonlyArray<string>;
  logoUrl: string | null;
}) {
  const pathname = usePathname();
  const tabs = deriveVisibleTabs(permissions);

  return (
    <nav className="border-b border-gray-6 bg-background" role="navigation" aria-label="Portal navigation">
      <div className="mx-auto flex h-14 max-w-5xl items-center gap-6 px-4">
        {logoUrl ? (
          <img
            src={logoUrl}
            alt="Logo"
            className="h-8 w-auto"
          />
        ) : null}

        <div className="flex items-center gap-1" role="tablist">
          {tabs.map((tab) => {
            const isActive = pathname.startsWith(tab.href);
            return (
              <Link
                key={tab.id}
                href={tab.href}
                role="tab"
                aria-selected={isActive}
                className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                  isActive
                    ? "bg-gray-3 text-gray-12"
                    : "text-gray-11 hover:bg-gray-3 hover:text-gray-12"
                }`}
              >
                {tab.label}
              </Link>
            );
          })}
        </div>
      </div>
    </nav>
  );
}
