type PortalHeaderProps = {
  logoUrl?: string;
  returnUrl?: string;
  appName?: string;
};

export function PortalHeader({ logoUrl, returnUrl, appName }: PortalHeaderProps) {
  return (
    <header className="w-full bg-[var(--portal-header-bg)] text-[var(--portal-header-fg)]">
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
              className="min-w-0 max-w-[50%] truncate rounded-md py-1.5 text-[var(--portal-header-muted)] text-sm transition-colors hover:text-[var(--portal-header-fg)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--portal-header-fg)] focus-visible:ring-offset-1 focus-visible:ring-offset-[var(--portal-header-bg)] sm:col-start-3 sm:row-start-1 sm:max-w-xs sm:justify-self-end"
            >
              <span aria-hidden="true">← </span>
              Return to {appName ?? "application"}
            </a>
          )}
        </div>
      </div>
    </header>
  );
}
