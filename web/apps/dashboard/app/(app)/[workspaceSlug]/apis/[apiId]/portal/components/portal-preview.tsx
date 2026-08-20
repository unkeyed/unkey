"use client";

import { isHexColor } from "@/lib/portal/validation";
import { cn } from "@/lib/utils";
import { onPrimaryColor } from "@unkey/ui/src/lib/branding";
import { useState } from "react";
import { DEFAULT_BRAND_COLOR, type PortalBrandingValue } from "./portal-branding";

/**
 * The real address is derived from the deployment's `portal_base_url`, which the
 * dashboard is not told, so the mock address bar stays generic rather than
 * asserting a host that would be wrong on staging or a self-hosted install.
 */
const MOCK_ADDRESS = "Your customer portal";

/**
 * Static, deliberately-lo-fi mock of the end-user portal page so operators can
 * see their logo + brand color in context before going live. Mirrors the real
 * portal layout (web/apps/portal): brand-colored header bar with the logo on
 * it, then the keys heading with "Create key" at the top of the list.
 *
 * The brand bar shows the slug because that is what the live portal renders:
 * `web/apps/portal/src/routes/_portal.tsx` passes `appName={portal?.slug}`.
 */
export function PortalPreview({
  slug,
  branding,
  className,
}: {
  slug: string;
  branding: PortalBrandingValue;
  className?: string;
}) {
  const [erroredUrl, setErroredUrl] = useState<string | null>(null);
  const color = isHexColor(branding.primaryColor) ? branding.primaryColor : DEFAULT_BRAND_COLOR;
  // The same helper the portal itself uses, so the preview cannot disagree with
  // what an end user sees.
  const onColor = onPrimaryColor(color);
  const showLogo = branding.logoUrl.trim().length > 0 && erroredUrl !== branding.logoUrl;

  return (
    <div
      className={cn(
        "flex w-full flex-col overflow-hidden rounded-lg border border-grayA-4 bg-gray-1 shadow-sm",
        className,
      )}
    >
      <div className="flex items-center gap-2 border-b border-grayA-3 bg-gray-2 px-3 py-2">
        <div className="flex gap-1.5">
          <span className="size-2 rounded-full bg-grayA-5" />
          <span className="size-2 rounded-full bg-grayA-5" />
          <span className="size-2 rounded-full bg-grayA-5" />
        </div>
        <div className="flex-1 truncate rounded-md border border-grayA-3 bg-gray-1 px-2 py-0.5 text-center text-[10px] text-gray-9">
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
              src={branding.logoUrl}
              alt=""
              onError={() => setErroredUrl(branding.logoUrl)}
              className="size-6 shrink-0 rounded-md object-contain"
            />
          )}
          <span className="truncate text-[13px] font-semibold" style={{ color: onColor }}>
            {slug}
          </span>
        </div>
        <span className="h-2 w-14 shrink-0 rounded" style={{ backgroundColor: `${onColor}66` }} />
      </div>

      <div className="flex flex-1 flex-col gap-3 px-4 py-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex flex-col gap-2">
            <div className="h-3 w-24 rounded bg-grayA-5" />
            <div className="h-2 w-44 max-w-full rounded bg-grayA-3" />
          </div>
          <div
            className="shrink-0 rounded-md px-3 py-1.5 text-[11px] font-medium"
            style={{ backgroundColor: color, color: onColor }}
          >
            <div className="h-2 bg-background/20 rounded-sm w-8" />
          </div>
        </div>
        <div className="rounded-lg border border-grayA-3">
          {[0, 1, 2, 3].map((row) => (
            <div
              key={row}
              className={cn(
                "flex items-center justify-between px-3 py-3",
                row > 0 && "border-t border-grayA-3",
              )}
            >
              <div className="flex flex-col gap-1.5">
                <div className="h-2 w-20 rounded bg-grayA-5" />
                <div className="h-1.5 w-32 rounded bg-grayA-3" />
              </div>
              <div className="h-2 w-10 rounded bg-grayA-3" />
            </div>
          ))}
        </div>
      </div>

      <div className="border-t border-grayA-3 px-4 py-2 text-center text-[10px] text-gray-8">
        Powered by Unkey
      </div>
    </div>
  );
}
