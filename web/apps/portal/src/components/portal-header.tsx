type PortalHeaderProps = {
  logoUrl?: string;
  returnUrl?: string;
  appName?: string;
};

/**
 * Branded portal header: a colored bar (customer's primary color, dark
 * fallback) carrying the logo on the left and a return-to-application link on
 * the right. Tabbed navigation was removed while Analytics and Docs are
 * deferred to v2; the portal currently exposes only the Keys page.
 */
export function PortalHeader({ logoUrl, returnUrl, appName }: PortalHeaderProps) {
  return (
    <header
      className="w-full text-[var(--portal-primary-foreground,#ffffff)]"
      style={{ backgroundColor: "var(--portal-primary, var(--color-gray-12))" }}
    >
      <div className="flex h-14 items-center justify-between gap-6 px-4 sm:px-8">
        {(logoUrl || appName) && (
          <div className="flex items-center gap-2.5">
            {logoUrl && <img src={logoUrl} alt="" className="h-6 w-auto" aria-hidden="true" />}
            {appName && <span className="font-medium text-sm">{appName}</span>}
          </div>
        )}
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
