import type { Branding } from "~/routes/dave-initial-design/-seed";

export function PortalLogoHeader({ branding }: { branding: Branding }) {
  return (
    <header className="w-full text-white" style={{ backgroundColor: "var(--portal-bg)" }}>
      <div className="flex h-14 items-center justify-between px-8">
        <div className="flex items-center gap-3">
          {branding.logoUrl ? <img src={branding.logoUrl} alt="" className="h-6 w-auto" /> : null}
          <span className="font-medium text-sm">{branding.appName}</span>
        </div>
        <a href="/" className="text-sm text-white/85 transition-colors hover:text-white">
          ← Return to {branding.appName}
        </a>
      </div>
    </header>
  );
}
