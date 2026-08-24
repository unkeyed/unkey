"use client";

import { isHexColor, logoUrlSchema } from "@/lib/portal/validation";
import { cn } from "@/lib/utils";
import { onPrimaryColor } from "@unkey/ui/src/lib/branding";
import { useState } from "react";
import { DEFAULT_BRAND_COLOR, type PortalBrandingValue } from "./portal-branding";

// Generic rather than a real host: the dashboard is not told the deployment's
// `portal_base_url`.
const MOCK_ADDRESS = "Your customer portal";

// Literal hex, fixed light: the dashboard's Tailwind sets `theme.colors` rather
// than `theme.extend.colors`, so palette classes do not resolve here, and the
// portal ships light-only.
const LIGHT = {
  surface: "#ffffff",
  chrome: "#f5f5f5",
  border: "#e5e5e5",
  mutedText: "#a3a3a3",
  skeleton: "#d4d4d4",
  skeletonWeak: "#e5e5e5",
} as const;

/**
 * Static mock of the end-user portal so operators can see their logo and brand
 * color in context. The brand bar shows the display name because that is what
 * the live portal renders (web/apps/portal/src/routes/_portal.tsx).
 */
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
  // Validated at the sink, so a caller that forgets cannot make this fetch
  // an arbitrary URL.
  const logoUrl = branding.logoUrl.trim();
  const showLogo =
    logoUrl.length > 0 && logoUrlSchema.safeParse(logoUrl).success && erroredUrl !== logoUrl;

  return (
    <div
      // Border width comes from classes, ahead of `className`, so a caller can
      // still drop an edge with `border-b-0`; only the color has to be inline.
      className={cn(
        "flex w-full flex-col overflow-hidden rounded-lg border border-solid shadow-sm",
        className,
      )}
      // `colorScheme` keeps UA-styled descendants light too, so the mock cannot
      // pick up the dashboard's dark scheme.
      style={{
        colorScheme: "light",
        backgroundColor: LIGHT.surface,
        borderColor: LIGHT.border,
      }}
    >
      <div
        className="flex items-center gap-2 px-3 py-2"
        style={{ backgroundColor: LIGHT.chrome, borderBottom: `1px solid ${LIGHT.border}` }}
      >
        <div className="flex gap-1.5">
          {[0, 1, 2].map((dot) => (
            <span
              key={dot}
              className="size-2 rounded-full"
              style={{ backgroundColor: LIGHT.skeleton }}
            />
          ))}
        </div>
        <div
          className="flex-1 truncate rounded-md px-2 py-0.5 text-center text-[10px]"
          style={{
            backgroundColor: LIGHT.surface,
            border: `1px solid ${LIGHT.border}`,
            color: LIGHT.mutedText,
          }}
        >
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
          <span className="truncate text-[13px] font-semibold" style={{ color: onColor }}>
            {displayName}
          </span>
        </div>
        <span className="h-2 w-14 shrink-0 rounded" style={{ backgroundColor: `${onColor}66` }} />
      </div>

      <div className="flex flex-1 flex-col gap-3 px-4 py-4">
        <div className="flex items-start justify-between gap-3">
          <div className="flex flex-col gap-2">
            <div className="h-3 w-24 rounded" style={{ backgroundColor: LIGHT.skeleton }} />
            <div
              className="h-2 w-44 max-w-full rounded"
              style={{ backgroundColor: LIGHT.skeletonWeak }}
            />
          </div>
          <div
            className="shrink-0 rounded-md px-3 py-1.5 text-[11px] font-medium"
            style={{ backgroundColor: color }}
          >
            {/* Sits on the brand color, so it tints with the computed foreground. */}
            <div className="h-2 w-8 rounded-sm" style={{ backgroundColor: `${onColor}33` }} />
          </div>
        </div>
        <div className="rounded-lg" style={{ border: `1px solid ${LIGHT.border}` }}>
          {[0, 1, 2, 3].map((row) => (
            <div
              key={row}
              className="flex items-center justify-between px-3 py-3"
              style={row > 0 ? { borderTop: `1px solid ${LIGHT.border}` } : undefined}
            >
              <div className="flex flex-col gap-1.5">
                <div className="h-2 w-20 rounded" style={{ backgroundColor: LIGHT.skeleton }} />
                <div
                  className="h-1.5 w-32 rounded"
                  style={{ backgroundColor: LIGHT.skeletonWeak }}
                />
              </div>
              <div className="h-2 w-10 rounded" style={{ backgroundColor: LIGHT.skeletonWeak }} />
            </div>
          ))}
        </div>
      </div>

      <div
        className="px-4 py-2 text-center text-[10px]"
        style={{ borderTop: `1px solid ${LIGHT.border}`, color: LIGHT.mutedText }}
      >
        Powered by Unkey
      </div>
    </div>
  );
}
