import { Link, useLocation } from "@tanstack/react-router";
import { ChartColumn, KeyRound, type LucideIcon } from "lucide-react";
import { type PortalTab, deriveVisibleTabs } from "~/lib/scopes";

const TAB_ICONS: Record<PortalTab["id"], LucideIcon> = {
  keys: KeyRound,
  analytics: ChartColumn,
};

type PortalHeaderProps = {
  scopes: ReadonlyArray<string>;
  logoUrl?: string;
  returnUrl?: string;
  appName?: string;
};

/**
 * Branded portal header: a colored bar (customer's primary color, dark
 * fallback) with the logo on the left, scope-derived navigation centered and a
 * return-to-application link on the right. Below `sm` it stacks to two rows.
 */
export function PortalHeader({ scopes, logoUrl, returnUrl, appName }: PortalHeaderProps) {
  const pathname = useLocation({ select: (location) => location.pathname });
  const tabs = deriveVisibleTabs(scopes);

  return (
    <header
      className="w-full text-[var(--portal-primary-foreground,#ffffff)]"
      style={{ backgroundColor: "var(--portal-primary, var(--color-gray-12))" }}
    >
      <div className="flex min-h-14 flex-col gap-2 px-4 py-3 sm:grid sm:grid-cols-[1fr_auto_1fr] sm:items-center sm:gap-6 sm:px-8 sm:py-0">
        <div className="flex min-w-0 items-center justify-between gap-6 sm:contents">
          {(logoUrl || appName) && (
            <div className="flex min-w-0 items-center gap-2.5 sm:max-w-sm">
              {logoUrl && <img src={logoUrl} alt="" className="h-6 w-auto" aria-hidden="true" />}
              {appName && (
                <span className="truncate font-medium text-sm" title={appName}>
                  {appName}
                </span>
              )}
            </div>
          )}
          {returnUrl && (
            <a
              href={returnUrl}
              title={`Return to ${appName ?? "application"}`}
              className="min-w-0 max-w-[50%] truncate rounded-md py-1.5 text-[color-mix(in_srgb,var(--portal-primary-foreground,#ffffff)_85%,transparent)] text-sm transition-colors hover:text-[var(--portal-primary-foreground,#ffffff)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--portal-primary-foreground,#ffffff)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--portal-primary,var(--color-gray-12))] sm:col-start-3 sm:row-start-1 sm:max-w-xs sm:justify-self-end"
            >
              <span aria-hidden="true">← </span>
              Return to {appName ?? "application"}
            </a>
          )}
        </div>
        {tabs.length > 1 && (
          <nav
            className="-mx-3 flex items-center gap-1 sm:col-start-2 sm:row-start-1 sm:mx-0 sm:justify-self-center"
            aria-label="Portal navigation"
          >
            {tabs.map((tab) => {
              const isActive = pathname.startsWith(tab.href);
              const Icon = TAB_ICONS[tab.id];
              return (
                <Link
                  key={tab.id}
                  to={tab.href}
                  aria-current={isActive ? "page" : undefined}
                  className={`flex items-center gap-2 rounded-md px-3 py-1.5 font-medium text-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--portal-primary-foreground,#ffffff)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--portal-primary,var(--color-gray-12))] ${
                    isActive
                      ? "bg-[color-mix(in_srgb,var(--portal-primary-foreground,#ffffff)_15%,transparent)]"
                      : "text-[color-mix(in_srgb,var(--portal-primary-foreground,#ffffff)_80%,transparent)] hover:bg-[color-mix(in_srgb,var(--portal-primary-foreground,#ffffff)_10%,transparent)] hover:text-[var(--portal-primary-foreground,#ffffff)]"
                  }`}
                >
                  <Icon className="size-4" aria-hidden="true" />
                  {tab.label}
                </Link>
              );
            })}
          </nav>
        )}
      </div>
    </header>
  );
}
