import { Link, useLocation } from "@tanstack/react-router";
import { deriveVisibleTabs } from "~/lib/permissions";

type PortalHeaderProps = {
  permissions: string[];
  logoUrl?: string;
  returnUrl?: string;
  appName?: string;
};

/**
 * Branded portal header: a colored bar (customer's primary color, dark
 * fallback) carrying the logo and permission-derived navigation on the left and
 * a return-to-application link on the right.
 */
export function PortalHeader({ permissions, logoUrl, returnUrl, appName }: PortalHeaderProps) {
  const location = useLocation();
  const tabs = deriveVisibleTabs(permissions);

  return (
    <header
      className="w-full text-[var(--portal-primary-foreground,#ffffff)]"
      style={{ backgroundColor: "var(--portal-primary, var(--color-gray-12))" }}
    >
      <div className="flex h-14 items-center justify-between gap-6 px-4 sm:px-8">
        <div className="flex items-center gap-6">
          {(logoUrl || appName) && (
            <div className="flex items-center gap-2.5">
              {logoUrl && <img src={logoUrl} alt="" className="h-6 w-auto" aria-hidden="true" />}
              {appName && <span className="font-medium text-sm">{appName}</span>}
            </div>
          )}
          <nav className="flex items-center gap-1" aria-label="Portal navigation">
            {tabs.map((tab) => {
              const isActive = location.pathname.startsWith(tab.href);
              return (
                <Link
                  key={tab.id}
                  to={tab.href}
                  aria-current={isActive ? "page" : undefined}
                  className={`rounded-md px-3 py-1.5 font-medium text-sm transition-colors ${
                    isActive
                      ? "bg-[color-mix(in_srgb,var(--portal-primary-foreground,#ffffff)_15%,transparent)]"
                      : "text-[color-mix(in_srgb,var(--portal-primary-foreground,#ffffff)_80%,transparent)] hover:bg-[color-mix(in_srgb,var(--portal-primary-foreground,#ffffff)_10%,transparent)] hover:text-[var(--portal-primary-foreground,#ffffff)]"
                  }`}
                >
                  {tab.label}
                </Link>
              );
            })}
          </nav>
        </div>
        {returnUrl && (
          <a
            href={returnUrl}
            className="whitespace-nowrap text-[color-mix(in_srgb,var(--portal-primary-foreground,#ffffff)_85%,transparent)] text-sm transition-colors hover:text-[var(--portal-primary-foreground,#ffffff)]"
          >
            ← Return to {appName ?? "application"}
          </a>
        )}
      </div>
    </header>
  );
}
