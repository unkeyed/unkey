import { Link, useLocation } from "@tanstack/react-router";
import { deriveVisibleTabs } from "~/lib/scopes";

type PortalHeaderProps = {
  scopes: ReadonlyArray<string>;
  logoUrl?: string;
  returnUrl?: string;
  appName?: string;
};

/**
 * Branded portal header: a colored bar (customer's primary color, dark
 * fallback) carrying the logo and scope-derived navigation on the left and a
 * return-to-application link on the right.
 */
export function PortalHeader({ scopes, logoUrl, returnUrl, appName }: PortalHeaderProps) {
  const pathname = useLocation({ select: (location) => location.pathname });
  const tabs = deriveVisibleTabs(scopes);

  const returnLink = (label: string, className: string) =>
    returnUrl && (
      <a
        href={returnUrl}
        className={`whitespace-nowrap text-[color-mix(in_srgb,var(--portal-primary-foreground,#ffffff)_85%,transparent)] text-sm transition-colors hover:text-[var(--portal-primary-foreground,#ffffff)] ${className}`}
      >
        ← {label}
      </a>
    );

  return (
    <header
      className="w-full text-[var(--portal-primary-foreground,#ffffff)]"
      style={{ backgroundColor: "var(--portal-primary, var(--color-gray-12))" }}
    >
      <div className="flex min-h-14 flex-col gap-2 px-4 py-3 sm:flex-row sm:items-center sm:gap-6 sm:px-8 sm:py-0">
        <div className="flex items-center justify-between gap-6">
          {(logoUrl || appName) && (
            <div className="flex items-center gap-2.5">
              {logoUrl && <img src={logoUrl} alt="" className="h-6 w-auto" aria-hidden="true" />}
              {appName && <span className="font-medium text-sm">{appName}</span>}
            </div>
          )}
          {returnLink("Go back", "sm:hidden")}
        </div>
        {tabs.length > 1 && (
          <nav
            className="-mx-3 flex items-center gap-1 sm:mx-0 sm:mr-auto"
            aria-label="Portal navigation"
          >
            {tabs.map((tab) => {
              const isActive = pathname.startsWith(tab.href);
              return (
                <Link
                  key={tab.id}
                  to={tab.href}
                  aria-current={isActive ? "page" : undefined}
                  className={`rounded-md px-3 py-1.5 font-medium text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--portal-primary-foreground,#ffffff)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--portal-primary,var(--color-gray-12))] ${
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
        )}
        {returnLink(`Return to ${appName ?? "application"}`, "hidden sm:inline")}
      </div>
    </header>
  );
}
