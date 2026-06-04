import { LegacyAppRedirect } from "../_components/legacy-app-redirect";

// Bare /apps has no content of its own; forward to the default app's deployments,
// matching how /projects/[projectId] resolves. Without this page the segment falls
// through to the [...legacyTab] catch-all and redirects into an /apps/apps loop.
export default function AppsIndexPage() {
  return <LegacyAppRedirect suffix="deployments" />;
}
