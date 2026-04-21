"use client";

import { deriveVisibleTabs } from "@/lib/permissions";
import Link from "next/link";
import { usePathname } from "next/navigation";

type PortalHeaderProps = {
  permissions: ReadonlyArray<string>;
  logoUrl: string | null;
  preview: boolean;
};

/**
 * Branded portal header with logo and permission-derived navigation tabs.
 * The logo comes from portal_branding; tabs are derived from session permissions.
 */
export function PortalHeader({ permissions, logoUrl, preview }: PortalHeaderProps) {
  const pathname = usePathname();
  const tabs = deriveVisibleTabs(permissions);

  return (
    <header className="border-b border-gray-6 bg-background">
      {preview ? <PreviewBannerInline /> : null}
      <div className="mx-auto flex h-14 max-w-5xl items-center gap-6 px-4">
        {logoUrl ? <img src={logoUrl} alt="Logo" className="h-8 w-auto" /> : null}

        <nav className="flex items-center gap-1" aria-label="Portal navigation">
          {tabs.map((tab) => {
            const isActive = pathname.startsWith(tab.href);
            return (
              <Link
                key={tab.id}
                href={tab.href}
                role="tab"
                aria-selected={isActive}
                className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                  isActive ? "text-gray-12" : "text-gray-11 hover:bg-gray-3 hover:text-gray-12"
                }`}
                style={
                  isActive
                    ? {
                        backgroundColor:
                          "color-mix(in srgb, var(--portal-primary, #6366f1) 10%, transparent)",
                        color: "var(--portal-primary, #6366f1)",
                      }
                    : undefined
                }
              >
                {tab.label}
              </Link>
            );
          })}
        </nav>
      </div>
    </header>
  );
}

/** Inline preview banner rendered inside the header when session has preview: true */
function PreviewBannerInline() {
  return (
    <output
      className="block bg-warning-3 border-b border-warning-6 px-4 py-1.5 text-center text-sm font-medium text-warning-11"
      aria-label="Preview mode active"
    >
      Preview mode — you are viewing this portal as an end user
    </output>
  );
}
