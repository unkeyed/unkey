"use client";

import { isHexColor, logoUrlSchema } from "@/lib/portal/validation";
import { cn } from "@/lib/utils";
import { onPrimaryColor } from "@unkey/ui/src/lib/branding";
import { useState } from "react";
import { DEFAULT_BRAND_COLOR, type PortalBrandingValue } from "./portal-branding";

// Generic because the dashboard is not told the deployment's `portal_base_url`.
const MOCK_ADDRESS = "Your customer portal";

// Static mock of the end-user portal. The brand bar shows the display name
// because that is what the live portal renders in
// `web/apps/portal/src/routes/_portal.tsx`.
export function PortalPreview({
  displayName,
  branding,
  className,
}: {
  displayName: string;
  branding: PortalBrandingValue;
  className?: string;
}) {
  const [erroredUrl, setErroredUrl] = useState<string | null>(null);
  const color = isHexColor(branding.primaryColor) ? branding.primaryColor : DEFAULT_BRAND_COLOR;
  // The same helper the portal itself uses, so the two cannot disagree.
  const onColor = onPrimaryColor(color);
  // Validated at the sink so a caller cannot make this `<img>` fetch an
  // arbitrary URL.
  const logoUrl = branding.logoUrl.trim();
  const showLogo =
    logoUrl.length > 0 && logoUrlSchema.safeParse(logoUrl).success && erroredUrl !== logoUrl;

  return (
    <div
      className={cn(
        "flex w-full flex-col overflow-hidden rounded-lg border border-solid border-[hsl(0_0%_91%)] bg-[hsl(0_0%_99%)] shadow-sm",
        className,
      )}
      style={{ colorScheme: "light" }}
    >
      <div className="flex items-center gap-2 border-b border-[hsl(0_0%_91%)] bg-[hsl(0_0%_94%)] px-3 py-2">
        <div className="flex gap-1.5">
          {[0, 1, 2].map((dot) => (
            <span key={dot} className="size-2 rounded-full bg-[hsl(0_0%_85%)]" />
          ))}
        </div>
        <div className="flex-1 truncate rounded-md border border-[hsl(0_0%_91%)] bg-[hsl(0_0%_99%)] px-2 py-0.5 text-center text-[10px] text-[hsl(0_0%_55%)]">
          {MOCK_ADDRESS}
        </div>
      </div>

      <div
        className="flex items-center justify-between px-4 py-3"
        style={{ backgroundColor: color }}
      >
        <div className="flex min-w-0 items-center gap-2.5">
          {showLogo && (
            <img
              src={logoUrl}
              alt=""
              onError={() => setErroredUrl(logoUrl)}
              className="size-6 shrink-0 rounded-md object-contain"
            />
          )}
          <span className="truncate text-[13px] font-[450]" style={{ color: onColor }}>
            {displayName}
          </span>
        </div>
        <span className="h-2 w-14 shrink-0 rounded" style={{ backgroundColor: `${onColor}66` }} />
      </div>

      <div className="flex flex-1 flex-col gap-3 px-4 py-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex flex-col gap-2">
            <div className="h-3 w-24 rounded bg-[hsl(0_0%_85%)]" />
            <div className="h-2 w-44 max-w-full rounded bg-[hsl(0_0%_91%)]" />
          </div>
          <div
            className="shrink-0 rounded-md px-3 py-1.5 text-[11px] font-normal"
            style={{ backgroundColor: color }}
          >
            <div className="h-2 w-8 rounded-sm" style={{ backgroundColor: `${onColor}33` }} />
          </div>
        </div>
        <div className="rounded-lg border border-[hsl(0_0%_91%)]">
          {[0, 1, 2, 3].map((row) => (
            <div
              key={row}
              className={cn(
                "flex items-center justify-between px-3 py-3",
                row > 0 && "border-t border-[hsl(0_0%_91%)]",
              )}
            >
              <div className="flex flex-col gap-1.5">
                <div className="h-2 w-20 rounded bg-[hsl(0_0%_85%)]" />
                <div className="h-1.5 w-32 rounded bg-[hsl(0_0%_91%)]" />
              </div>
              <div className="h-2 w-10 rounded bg-[hsl(0_0%_91%)]" />
            </div>
          ))}
        </div>
      </div>

      <div className="border-t border-[hsl(0_0%_91%)] px-4 py-2 text-center text-[10px] text-[hsl(0_0%_55%)]">
        Powered by Unkey
      </div>
    </div>
  );
}
