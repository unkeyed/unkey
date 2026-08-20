"use client";

import { isHexColor, logoUrlSchema } from "@/lib/portal/validation";
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
 * Literal hex rather than palette classes. The dashboard's Tailwind config sets
 * `theme.colors` instead of `theme.extend.colors`, which replaces the default
 * palette outright — `neutral-*` and even `white` do not resolve here, so those
 * classes would silently render nothing.
 */
const LIGHT = {
  surface: "#ffffff",
  chrome: "#f5f5f5",
  border: "#e5e5e5",
  mutedText: "#a3a3a3",
  skeleton: "#d4d4d4",
  skeletonWeak: "#e5e5e5",
} as const;

/**
 * Static, deliberately-lo-fi mock of the end-user portal page so operators can
 * see their logo + brand color in context before going live. Mirrors the real
 * portal layout (web/apps/portal): brand-colored header bar with the logo on
 * it, then the keys heading with "Create key" at the top of the list.
 *
 * The brand bar shows the slug because that is what the live portal renders:
 * `web/apps/portal/src/routes/_portal.tsx` passes `appName={portal?.slug}`.
 *
 * Every neutral here is a fixed light value rather than a `gray-*` token. The
 * portal ships light-only — its root sets no `dark` class and its source has no
 * `dark:` variants — so a preview built from dashboard tokens would turn dark
 * with the operator's own theme and show them a portal their users never see.
 * The brand color and its computed foreground are the only theme-varying parts,
 * because those are the parts the operator actually controls.
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
  // Validated here rather than trusting the caller: this is the sink that turns
  // the value into a request, so the scheme check travels with it.
  const logoUrl = branding.logoUrl.trim();
  const showLogo =
    logoUrl.length > 0 && logoUrlSchema.safeParse(logoUrl).success && erroredUrl !== logoUrl;

  return (
    <div
      className={cn("flex w-full flex-col overflow-hidden rounded-lg shadow-sm", className)}
      // Keeps any UA-styled descendant light too, so the mock cannot pick up the
      // dashboard's dark scheme.
      style={{
        colorScheme: "light",
        backgroundColor: LIGHT.surface,
        border: `1px solid ${LIGHT.border}`,
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
            {slug}
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
            {/* Sits on the brand color, so its tint comes from the computed
                foreground rather than a theme token. */}
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
