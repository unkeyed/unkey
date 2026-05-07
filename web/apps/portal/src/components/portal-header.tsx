import { Link, useLocation } from "@tanstack/react-router";
import { deriveVisibleTabs } from "~/lib/permissions";

type PortalHeaderProps = {
  permissions: string[];
  logoUrl?: string;
};

/**
 * Branded portal header with permission-derived navigation tabs.
 */
export function PortalHeader({ permissions, logoUrl }: PortalHeaderProps) {
  const location = useLocation();
  const tabs = deriveVisibleTabs(permissions);

  return (
    <header className="border-gray-6 border-b bg-background">
      <div className="mx-auto flex h-14 max-w-5xl items-center gap-6 px-4">
        {logoUrl && <img src={logoUrl} alt="" className="h-8 w-auto" aria-hidden="true" />}
        <nav className="flex items-center gap-1" aria-label="Portal navigation">
          {tabs.map((tab) => {
            const isActive = location.pathname.startsWith(tab.href);
            return (
              <Link
                key={tab.id}
                to={tab.href}
                aria-current={isActive ? "page" : undefined}
                className={`rounded-md px-3 py-1.5 font-medium text-sm transition-colors ${
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
