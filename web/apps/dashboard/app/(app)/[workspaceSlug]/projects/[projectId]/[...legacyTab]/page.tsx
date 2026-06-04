"use client";
import { useParams } from "next/navigation";
import { LegacyAppRedirect } from "../_components/legacy-app-redirect";

// Catch-all for pre-move project-scoped tab URLs (e.g. /projects/[id]/deployments).
// The static `apps/[appId]` segment takes precedence, so this only matches the
// legacy paths and forwards them to the default app.
export default function LegacyTabRedirect() {
  const params = useParams();
  const legacyTab = params?.legacyTab;
  const suffix = Array.isArray(legacyTab) ? legacyTab.join("/") : (legacyTab ?? "");
  return <LegacyAppRedirect suffix={suffix} />;
}
