import { brandingToCssVars, loadBranding } from "@/lib/branding";
import { resolvePortalConfig } from "@/lib/portal-config";
import "@unkey/ui/css";
import "@/styles/tailwind.css";
import { GeistMono } from "geist/font/mono";
import { GeistSans } from "geist/font/sans";
import type { Metadata } from "next";
import { headers } from "next/headers";
import type React from "react";

/** All portal pages are dynamic — they depend on hostname and session state */
export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Customer Portal",
  description: "Manage your API keys and view analytics",
  robots: { index: false, follow: false },
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const headersList = await headers();
  const hostname = headersList.get("host")?.split(":")[0] ?? "";

  const portalConfig = await resolvePortalConfig(hostname);

  const branding = portalConfig ? await loadBranding(portalConfig.id) : null;
  const cssVars = branding ? brandingToCssVars(branding) : {};

  return (
    <html
      lang="en"
      className={`${GeistSans.variable} ${GeistMono.variable}`}
      style={cssVars as React.CSSProperties}
    >
      <body className="min-h-screen bg-portal-secondary antialiased">
        {children}
      </body>
    </html>
  );
}
