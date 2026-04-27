import { Link, useLocation } from "@tanstack/react-router";
import { deriveVisibleTabs } from "~/lib/permissions";

/**
 * Branded portal header with permission-derived navigation tabs.
 * Tabs are derived from session permissions stored in the cookie.
 * For the PoC, all tabs are shown since permissions aren't decoded client-side yet.
 */
export function PortalHeader() {
  const location = useLocation();

  // For PoC, show all tabs. When the session token is decoded server-side,
  // permissions will be passed down via route context.
  const allPermissions = ["keys:read", "keys:write", "analytics:read", "docs:read"];
  const tabs = deriveVisibleTabs(allPermissions);

  return (
    <header className="border-b border-gray-6 bg-background">
      <div className="mx-auto flex h-14 max-w-5xl items-center gap-6 px-4">
        <nav className="flex items-center gap-1" aria-label="Portal navigation">
          {tabs.map((tab) => {
            const isActive = location.pathname.startsWith(tab.href);
            return (
              <Link
                key={tab.id}
                to={tab.href}
                aria-current={isActive ? "page" : undefined}
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
