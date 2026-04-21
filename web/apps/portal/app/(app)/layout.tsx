import { PortalHeader } from "@/components/portal-header";
import { loadBranding } from "@/lib/branding";
import { db } from "@/lib/db";
import { getSessionToken } from "@/lib/session";
import { and, eq, gt, schema } from "@unkey/db";
import { redirect } from "next/navigation";
import type React from "react";

/**
 * Shared layout for authenticated portal pages (keys, analytics, docs).
 * Reads the session cookie, loads permissions and branding, renders the header.
 * The root page (session exchange) is outside this route group and has no header.
 */
export default async function AppLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const token = await getSessionToken();
  if (!token) {
    redirect("/");
  }

  const now = Date.now();
  const session = await db.query.portalSessions.findFirst({
    where: and(eq(schema.portalSessions.id, token), gt(schema.portalSessions.expiresAt, now)),
  });

  if (!session) {
    // Session expired or invalid — redirect to root which handles error display
    redirect("/");
  }

  const branding = await loadBranding(session.portalConfigId);

  return (
    <>
      <PortalHeader
        permissions={session.permissions}
        logoUrl={branding.logoUrl}
        preview={session.preview}
      />
      {children}
    </>
  );
}
